# **Gemini.md: A Reproducible Development Environment for an LLM-driven O-RAN Copilot**

## **1\. Project Description**

This project, "Gemini Code CLI," provides a proof-of-concept for a software architecture copilot that translates natural language commands into declarative network intents. It integrates a Large Language Model (LLM) with Nephio R5 to automate O-RAN network slicing and Network Function (NF) management.

The core workflow is as follows:

1. A user issues a command like "Create a network slice for IoT devices in the west region."  
2. An LLM/RAG service, using a knowledge base of O-RAN and Nephio specifications, generates a corresponding Nephio KRM package.  
3. The package is committed to a Git repository, where Nephio's control plane discovers, specializes, validates, and approves it.  
4. Once published, an O-RAN SMO Adaptor translates the KRM intent into API calls over O1, O2, and A1 interfaces to configure the network.

This Gemini.md provides the complete setup to bootstrap a development and testing environment for this system.

## **2\. Project Folder Structure**

gemini-code-cli/  
├──.devcontainer/  
│ ├── devcontainer.json \# VSCode Dev Container configuration  
│ └── Dockerfile \# Dockerfile for the dev environment  
├── Makefile \# Top-level make targets for automation  
├── docs/ \# Project documentation  
│ └── architecture.md  
├── knowledge\_base/ \# Source documents for RAG  
│ ├── nephio\_r5/ \# Nephio R5 documentation PDFs and text files  
│ └── o-ran\_specs\_2025/ \# O-RAN specification PDFs  
├── packages/  
│ └── krm\_templates/ \# KRM blueprint templates  
│ └── network\_slice.yaml  
└── src/  
├── cli/ \# Main CLI entrypoint  
│ └── main.py  
├── llm\_service/ \# LLM/RAG intent generation service  
│ ├── agent.py  
│ ├── rag\_builder.py  
│ └── requirements.txt  
└── smo\_adapter/ \# O-RAN SMO Adaptor service  
├── go.mod  
└── main.go

\#\# 3\. Dev Container Configuration (\`.devcontainer/devcontainer.json\`)

This file configures a VSCode Dev Container to ensure a consistent and reproducible development environment.

\`\`\`json  
{  
  "name": "Gemini Code CLI Dev Environment",  
  "build": {  
    "dockerfile": "Dockerfile",  
    "context": ".."  
  },  
  "runArgs": \[  
    "--privileged",  
    "--name=gemini-code-cli-dev"  
    \],  
  "mounts": \[  
    "source=/var/run/docker.sock,target=/var/run/docker.sock,type=bind"  
  \],  
  "customizations": {  
    "vscode": {  
      "settings": {  
        "go.toolsManagement.autoUpdate": true,  
        "python.defaultInterpreterPath": "/usr/local/bin/python"  
      },  
      "extensions": \[  
        "ms-python.python",  
        "golang.go",  
        "ms-azuretools.vscode-docker",  
        "redhat.vscode-yaml",  
        "ms-kubernetes-tools.vscode-kubernetes-tools"  
      \]  
    }  
  },  
  "postCreateCommand": "make bootstrap"  
}

A corresponding .devcontainer/Dockerfile would be created to install Go 1.21+, Python 3.11+, Docker, kubectl, kind (v0.23.0+), and kpt (v1.0.0-beta+).

## **4\. Makefile Targets**

This Makefile automates common development tasks.

Makefile

\# Makefile for Gemini Code CLI

\# Variables  
KIND\_CLUSTER\_NAME := nephio-mgmt  
NEPHIO\_CATALOG\_REPO := \[https://github.com/nephio-project/catalog.git\](https://github.com/nephio-project/catalog.git)  
NEPHIO\_CATALOG\_TAG := origin/main  
PYTHON := python3

**.PHONY**: help bootstrap nephio-up test run clean

help:  
	@echo "Makefile for Gemini Code CLI"  
	@echo "Targets:"  
	@echo "  bootstrap    \- Install all dependencies and tools."  
	@echo "  nephio-up    \- Bootstrap a local Nephio R5 management cluster using KinD."  
	@echo "  test         \- Run all unit and integration tests."  
	@echo "  run          \- Start the Gemini Code CLI services (LLM service, SMO adapter)."  
	@echo "  clean        \- Tear down the Nephio cluster and clean up resources."

bootstrap:  
	\# Install Python dependencies  
	$(PYTHON) \-m pip install \--upgrade pip  
	$(PYTHON) \-m pip install \-r src/llm\_service/requirements.txt  
	\# Install Go dependencies  
	cd src/smo\_adapter && go mod tidy  
	\# Build RAG knowledge base  
	$(PYTHON) src/llm\_service/rag\_builder.py  
	@echo "Bootstrap complete."

nephio-up:  
	@echo "Bringing up Nephio R5 on KinD..."  
	\# 1\. Create KinD cluster  
	kind create cluster \--name $(KIND\_CLUSTER\_NAME)  
	\# 2\. Install Porch  
	kpt pkg get \--for-deployment $(NEPHIO\_CATALOG\_REPO)/nephio/core/porch@$(NEPHIO\_CATALOG\_TAG) porch  
	kpt fn render porch  
	kpt live init porch  
	kpt live apply porch \--reconcile-timeout=15m \--output=table  
	\# 3\. Install Nephio Controllers & Gitea Secret (requires user input)  
	kpt pkg get \--for-deployment $(NEPHIO\_CATALOG\_REPO)/nephio/core/nephio-operator@$(NEPHIO\_CATALOG\_TAG) nephio-operator  
	@echo "Please edit nephio-operator/secret.yaml with your Gitea credentials."  
	@read \-p "Press Enter to continue..."  
	kpt fn render nephio-operator  
	kpt live init nephio-operator  
	kpt live apply nephio-operator \--reconcile-timeout=15m \--output=table  
	\# 4\. Install Gitea, ConfigSync, and Stock Repos  
	\# (Further steps from Nephio docs would follow)  
	@echo "Nephio management cluster is ready."

test:  
	\# Run Python unit tests  
	$(PYTHON) \-m pytest src/llm\_service/  
	\# Run Go unit tests  
	cd src/smo\_adapter && go test./...  
	@echo "All tests passed."

run:  
	@echo "Starting Gemini Code CLI services..."  
	\# Start the LLM/RAG service (e.g., using FastAPI or gRPC)  
	\# Ensure environment variables for LLM API keys are set  
	$(PYTHON) src/cli/main.py &  
	\# Start the SMO Adapter  
	cd src/smo\_adapter && go run. &  
	@echo "Services running in background."

clean:  
	@echo "Cleaning up environment..."  
	kind delete cluster \--name $(KIND\_CLUSTER\_NAME)  
	@echo "Cleanup complete."

## **5\. KPT Package Template (packages/krm\_templates/network\_slice.yaml)**

This template serves as a structured "form" for the LLM to populate. The \# LLM\_PLACEHOLDER comments indicate fields the LLM must generate based on the user's natural language command and the RAG context.

YAML

\# packages/krm\_templates/network\_slice.yaml  
apiVersion: config.nephio.org/v1alpha1  
kind: PackageVariant  
metadata:  
  name: "\# LLM\_PLACEHOLDER: e.g., slice-enterprise-vr-paris"  
  namespace: default  
spec:  
  upstream:  
    repo: blueprints  
    package: o-ran-network-slice  
    revision: v1.0.0  
  downstream:  
    repo: "\# LLM\_PLACEHOLDER: e.g., deployment-repo-region-eu-west1"  
    package: "\# LLM\_PLACEHOLDER: e.g., slice-enterprise-vr-paris"  
  injectors:  
  \- group: workload.nephio.org  
    version: v1alpha1  
    kind: Slice  
    name: "\# LLM\_PLACEHOLDER: e.g., slice-enterprise-vr-paris-config"  
\---  
apiVersion: workload.nephio.org/v1alpha1  
kind: Slice  
metadata:  
  name: "\# LLM\_PLACEHOLDER: e.g., slice-enterprise-vr-paris-config"  
  namespace: default  
spec:  
  \# These parameters are generated by the LLM from the natural language command  
  \# and O-RAN specification context.  
  sNSSAI:  
    sst: "\# LLM\_PLACEHOLDER: e.g., 1 (for eMBB)"  
    sd: "\# LLM\_PLACEHOLDER: e.g., '000123'"  
  qualityOfService:  
    \# Based on O-RAN Slicing Architecture v13+  
    fiveQI: "\# LLM\_PLACEHOLDER: e.g., 83 (for VR/AR)"  
    arp:  
      priorityLevel: "\# LLM\_PLACEHOLDER: e.g., 7"  
      preemptionCapability: "MAY\_PREEMPT"  
      preemptionVulnerability: "NOT\_PREEMPTABLE"  
  maxDataBurstVolume: \# LLM\_PLACEHOLDER: e.g., 2000 (in MB)  
  maxNumberOfUEs: \# LLM\_PLACEHOLDER: e.g., 500  
  areaOfService:  
    \# This would be mapped to specific network elements (O-CUs/O-DUs) by Nephio functions  
    trackingAreaList:

## **6\. Sample RAG Pipeline (src/llm\_service/agent.py snippet)**

This illustrative Python snippet shows how LangChain can be used to create the RAG chain that powers the intent translation.

Python

\# src/llm\_service/agent.py (Illustrative Snippet)  
import os  
from langchain\_community.vectorstores import FAISS  
from langchain\_openai import OpenAIEmbeddings, ChatOpenAI  
from langchain.prompts import ChatPromptTemplate  
from langchain.schema.runnable import RunnablePassthrough  
from langchain.schema.output\_parser import StrOutputParser

\# Assume LlamaIndex has already built and saved the index in rag\_builder.py  
KNOWLEDGE\_BASE\_PATH \= "faiss\_index"

def get\_rag\_chain():  
    """Builds and returns the RAG chain."""  
      
    \# 1\. Load the pre-built knowledge base  
    \# For production, use a more robust vector store.  
    embeddings \= OpenAIEmbeddings()  
    vectorstore \= FAISS.load\_local(KNOWLEDGE\_BASE\_PATH, embeddings, allow\_dangerous\_deserialization=True)  
    retriever \= vectorstore.as\_retriever()

    \# 2\. Define the prompt template  
    template \= """  
You are an expert O-RAN and Nephio architect. Your task is to generate a valid Nephio KRM package based on the user's command.  
You must generate a 'PackageVariant' and its associated context resources.  
Use the following context from O-RAN and Nephio specifications to ensure the generated KRM is accurate and compliant.

Context: {context}

User Command: {command}

Generate only the YAML for the KRM package. Do not add any explanation.  
"""  
    prompt \= ChatPromptTemplate.from\_template(template)

    \# 3\. Initialize the LLM (replace with self-hosted Mistral/Llama endpoint)  
    model \= ChatOpenAI(model="gpt-4-turbo", temperature=0.0)

    \# 4\. Create the LangChain RAG chain  
    rag\_chain \= (  
        {"context": retriever, "command": RunnablePassthrough()}

| prompt  
| model  
| StrOutputParser()  
    )  
    return rag\_chain

def generate\_krm\_from\_command(command: str) \-\> str:  
    """  
    Takes a natural language command and returns a KRM YAML string.  
    """  
    print(f"Generating KRM for command: '{command}'")  
    chain \= get\_rag\_chain()  
    krm\_yaml \= chain.invoke(command)  
    \# TODO: Add a post-generation validation step using kpt or a Python KRM library  
    return krm\_yaml

\# Example Usage:  
if \_\_name\_\_ \== "\_\_main\_\_":  
    \# This would be called from the CLI entrypoint  
    user\_command \= "Create a high-bandwidth, low-latency slice for enterprise VR applications in the Paris site. Use sst 1 and sd 000123\. It needs to support 500 users and should be deployed to the 'deployment-repo-region-eu-west1' repository."  
    generated\_krm \= generate\_krm\_from\_command(user\_command)  
    print("--- Generated KRM \---")  
    print(generated\_krm)

\#\# 6\. Bibliography

\*   (\[https://docs.nephio.org/docs/\](https://docs.nephio.org/docs/), Nephio R5 official docs & release notes)  
\*   (\[https://docs.nephio.org/docs/release-notes/r5/\](https://docs.nephio.org/docs/release-notes/r5/), Nephio R5 Release Notes | Nephio Documentation)  
\*   (\[https://docs.nephio.org/docs/guides/user-guides/helm/flux-helm/\](https://docs.nephio.org/docs/guides/user-guides/helm/flux-helm/), Helm Integration in Nephio)  
\*   (\[https://docs.nephio.org/docs/porch/function-runner-pod-templates/\](https://docs.nephio.org/docs/porch/function-runner-pod-templates/), Function runner pod templating)  
\*   (\[https://lf-nephio.atlassian.net/\](https://lf-nephio.atlassian.net/), Nephio Jira)  
\*   (\[https://github.com/nephio-project/docs/blob/main/resources.md\](https://github.com/nephio-project/docs/blob/main/resources.md), Additional Resources)  
\*   (\[https://docs.nephio.org/docs/architecture/\](https://docs.nephio.org/docs/architecture/), Nephio Architecture | Nephio Documentation)  
\*   (\[https://docs.nephio.org/docs/guides/user-guides/\](https://docs.nephio.org/docs/guides/user-guides/), Nephio user guides)  
\*   (\[https://www.o-ran.org/blog/67-new-or-updated-o-ran-technical-documents-released-since-november-2024\](https://www.o-ran.org/blog/67-new-or-updated-o-ran-technical-documents-released-since-november-2024), 67 New or Updated O-RAN Technical Documents Released since November 2024\)  
\*   (\[https://www.o-ran.org/blog/69-new-or-updated-o-ran-technical-documents-released-since-november-2023\](https://www.o-ran.org/blog/69-new-or-updated-o-ran-technical-documents-released-since-november-2023), 69 New or Updated O-RAN Technical Documents Released since November 2023\)  
\*   (\[https://www.scribd.com/document/761828555/O-RAN-WG1-Slicing-Architecture-R003-v13-00\](https://www.scribd.com/document/761828555/O-RAN-WG1-Slicing-Architecture-R003-v13-00), O RAN \- WG1.Slicing Architecture R003 v13.00)  
\*   (\[https://www.etsi.org/deliver/etsi\_TS/104000\_104099/104041/11.00.00\_60/ts\_104041v110000p.pdf\](https://www.etsi.org/deliver/etsi\_TS/104000\_104099/104041/11.00.00\_60/ts\_104041v110000p.pdf), ETSI TS 104 041 V11.0.0 (2025-03))  
\*   (\[https://www.o-ran.org/blog/o-ran-alliance-security-update-2025\](https://www.o-ran.org/blog/o-ran-alliance-security-update-2025), O-RAN ALLIANCE Security Update 2025\)  
\*   (\[https://www.o-ran.org/ecosystem-resources\](https://www.o-ran.org/ecosystem-resources), O-RAN Ecosystem Resources)  
\*   (\[https://huggingface.co/meta-llama/Llama-2-70b\](https://huggingface.co/meta-llama/Llama-2-70b), Llama 2 70B Model Card)  
\*   (\[https://arxiv.org/abs/2503.19217\](https://arxiv.org/abs/2503.19217), LLM Benchmarking with LLaMA2: Evaluating Code Development Performance Across Multiple Programming Languages)  
\*   (\[https://mistral.ai/news/mixtral-8x22b\](https://mistral.ai/news/mixtral-8x22b), Mixtral 8x22B)  
\*   (\[https://milvus.io/ai-quick-reference/how-does-haystack-differ-from-other-search-frameworks-like-langchain-and-llamaindex\](https://milvus.io/ai-quick-reference/how-does-haystack-differ-from-other-search-frameworks-like-langchain-and-llamaindex), How does Haystack differ from other search frameworks like LangChain and LlamaIndex?)  
\*   (\[https://milvus.io/ai-quick-reference/what-are-the-differences-between-langchain-and-other-llm-frameworks-like-llamaindex-or-haystack\](https://milvus.io/ai-quick-reference/what-are-the-differences-between-langchain-and-other-llm-frameworks-like-llamaindex-or-haystack), What are the differences between LangChain and other LLM frameworks like LlamaIndex or Haystack?)  
\*   (\[https://medium.com/@heyamit10/llamaindex-vs-langchain-vs-haystack-4fa8b15138fd\](https://medium.com/@heyamit10/llamaindex-vs-langchain-vs-haystack-4fa8b15138fd), LlamaIndex vs. LangChain vs. Haystack)  
\*   (\[https://arxiv.org/abs/2507.18515\](https://arxiv.org/abs/2507.18515), A Deep Dive into Retrieval-Augmented Generation for Code Completion: Experience on WeChat)  
\*   (\[https://github.com/nephio-project/nephio\](https://github.com/nephio-project/nephio), nephio-project/nephio GitHub Repository)  
\*   (\[https://github.com/o-ran-sc\](https://github.com/o-ran-sc), O-RAN Software Community GitHub Organization)  
\*   (\[https://docs.o-ran-sc.org/projects/o-ran-sc-nonrtric-plt-a1policymanagementservice/en/latest/overview.html\](https://docs.o-ran-sc.org/projects/o-ran-sc-nonrtric-plt-a1policymanagementservice/en/latest/overview.html), O-RAN A1 Interface Overview)  
\*   (\[https://www.techplayon.com/what-o1-interface-in-open-ran/\](https://www.techplayon.com/what-o1-interface-in-open-ran/), What is O1 Interface in Open RAN?)  
\*   (\[https://www.starlingx.io/blog/starlingx-oran-o2-application/\](https://www.starlingx.io/blog/starlingx-oran-o2-application/), StarlingX as an O-RAN O-Cloud with O2 Application)  
\*   (\[https://docs.o-ran-sc.org/en/latest/architecture/architecture.html\](https://docs.o-ran-sc.org/en/latest/architecture/architecture.html), O-RAN SC Architecture)  
