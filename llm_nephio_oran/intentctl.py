"""intentctl — CLI for the Nephoran Intent Pipeline.

Usage:
    intentctl create "Deploy eMBB slice with KPIMON"        # plan only
    intentctl create "Scale AMF to 3" --llm                 # use LLM planner
    intentctl run "Deploy eMBB slice" --llm                 # full pipeline: plan → generate → git → PR
    intentctl push schemas/intent-plan.generated.json       # push existing plan to intentd
"""
from __future__ import annotations

import json
import logging
import os
import tempfile
from pathlib import Path

import requests
import typer
from rich import print
from rich.panel import Panel

from llm_nephio_oran.validators.schema_validate import validate_json_file, validate_json_instance

app = typer.Typer(no_args_is_help=True, add_completion=False)
logger = logging.getLogger(__name__)


@app.command()
def create(
    text: str = typer.Argument(..., help="Natural language intent, e.g. 'Deploy eMBB slice'."),
    out: Path = typer.Option(Path("schemas/intent-plan.generated.json"), help="Output plan JSON path."),
    validate: bool = typer.Option(True, help="Validate plan against schema."),
    llm: bool = typer.Option(False, "--llm", help="Use Ollama LLM planner (default: stub)."),
):
    """Create an IntentPlan from natural language text."""
    from llm_nephio_oran.planner.llm_planner import plan_from_text
    plan = plan_from_text(text, use_llm=llm)
    out.write_text(json.dumps(plan, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(f"[green]Wrote[/green] {out}")
    if validate:
        validate_json_file("schemas/intent-plan.schema.json", out)
        print("[green]Schema validation OK[/green]")


@app.command()
def run(
    text: str = typer.Argument(..., help="Natural language intent — triggers full pipeline."),
    llm: bool = typer.Option(True, "--llm/--stub", help="Use LLM planner (default) or stub."),
    dry_run: bool = typer.Option(False, "--dry-run", help="Generate packages but don't push to Git."),
    out_dir: Path = typer.Option(Path("packages/instances"), help="Output directory for generated packages."),
):
    """Full pipeline: NL → plan → validate → generate kpt packages → git commit → PR."""
    # Step 1: Plan
    print(Panel(f"[bold]Intent:[/bold] {text}", title="Step 1: Planning"))
    from llm_nephio_oran.planner.llm_planner import plan_from_text
    plan = plan_from_text(text, use_llm=llm)
    intent_id = plan["intentId"]
    print(f"  intentId: [cyan]{intent_id}[/cyan]")
    print(f"  intentType: [cyan]{plan['intentType']}[/cyan]")
    for i, a in enumerate(plan.get("actions", [])):
        print(f"  action[{i}]: {a['kind']} {a['component']}" + (f" replicas={a['replicas']}" if "replicas" in a else ""))

    # Step 2: Validate
    print(Panel("Validating against schema", title="Step 2: Validation"))
    try:
        validate_json_instance("schemas/intent-plan.schema.json", plan)
        print("[green]  Schema validation: PASS[/green]")
    except ValueError as e:
        print(f"[red]  Schema validation: FAIL[/red]\n{e}")
        raise typer.Exit(code=1)

    # Step 3: Generate packages
    print(Panel("Generating kpt/KRM packages", title="Step 3: Generate"))
    from llm_nephio_oran.generator.kpt_generator import generate_kpt_packages
    pkg_dir = generate_kpt_packages(plan, out_dir)
    file_count = sum(1 for _ in pkg_dir.rglob("*") if _.is_file())
    print(f"  Generated [cyan]{file_count}[/cyan] files at [cyan]{pkg_dir}[/cyan]")

    # Step 4: Validate generated YAMLs
    print(Panel("Validating generated manifests", title="Step 4: Manifest Validation"))
    from llm_nephio_oran.validators.manifest_validator import validate_package
    errors = validate_package(pkg_dir)
    if errors:
        for err in errors:
            print(f"  [red]{err}[/red]")
        print(f"[red]  Manifest validation: {len(errors)} errors[/red]")
        raise typer.Exit(code=1)
    print("[green]  Manifest validation: PASS[/green]")

    if dry_run:
        print(Panel("[yellow]Dry run — skipping git push and PR creation[/yellow]", title="Step 5: GitOps (skipped)"))
        # Write plan to disk for reference
        plan_out = Path("schemas/intent-plan.generated.json")
        plan_out.write_text(json.dumps(plan, indent=2, ensure_ascii=False) + "\n")
        print(f"  Plan saved to [cyan]{plan_out}[/cyan]")
        return

    # Step 5: Git commit + PR
    print(Panel("Pushing to Gitea and creating PR", title="Step 5: GitOps"))
    from llm_nephio_oran.gitops.git_ops import push_to_gitops
    result = push_to_gitops(pkg_dir, plan)
    print(f"  Branch: [cyan]{result['branch']}[/cyan]")
    print(f"  Commit: [cyan]{result['commit_sha'][:8]}[/cyan]")
    pr = result.get("pr", {})
    if pr and pr.get("number"):
        print(f"  PR: [green]#{pr['number']}[/green] — {pr.get('html_url', 'N/A')}")
    elif pr and pr.get("dry_run"):
        print("  PR: [yellow]skipped (no GITEA_TOKEN)[/yellow]")

    # Step 6: Porch lifecycle (draft → proposed → published)
    print(Panel("Submitting to Porch lifecycle", title="Step 6: Porch"))
    try:
        from llm_nephio_oran.porch.client import PorchClient
        porch = PorchClient()
        pkg_name = f"instances/{intent_id}"
        draft = porch.create_draft(repo="nephoran-packages", package_name=pkg_name)
        draft_name = draft["metadata"]["name"]
        print(f"  Draft: [cyan]{draft_name}[/cyan]")

        # Push KRM resources into draft
        resources = {}
        for f in pkg_dir.rglob("*.yaml"):
            resources[str(f.relative_to(pkg_dir))] = f.read_text()
        porch.update_resources(draft_name, resources)
        print(f"  Resources: [cyan]{len(resources)} files pushed[/cyan]")

        # Propose for review
        porch.propose(draft_name)
        print(f"  Status: [yellow]Proposed[/yellow] (awaiting approval)")
    except Exception as e:
        print(f"  [yellow]Porch integration skipped: {e}[/yellow]")

    print()
    print(f"[bold green]Pipeline complete for {intent_id}[/bold green]")


@app.command()
def push(
    plan_path: Path = typer.Argument(..., exists=True, help="IntentPlan JSON file to submit to intentd."),
    base_url: str = typer.Option("http://localhost:8080", help="intentd base URL"),
):
    """Push an existing IntentPlan JSON to the intentd API server."""
    payload = json.loads(plan_path.read_text(encoding="utf-8"))
    r = requests.post(f"{base_url}/internal/validate/intent-plan", json=payload, timeout=10)
    r.raise_for_status()
    print("[green]Plan validated by server[/green]")
    r2 = requests.post(
        f"{base_url}/tmf-api/intent/v5/intent",
        json={"payload": payload},
        timeout=10,
    )
    r2.raise_for_status()
    print(r2.json())


def main():
    app()


if __name__ == "__main__":
    main()
