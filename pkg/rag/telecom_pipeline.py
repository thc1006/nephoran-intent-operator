from typing import Dict, Any
from langchain_community.vectorstores import Weaviate
from langchain_openai import ChatOpenAI, OpenAIEmbeddings
from langchain.chains import RetrievalQA
from langchain.prompts import PromptTemplate
import json


class TelecomRAGPipeline:
    """Production-ready RAG pipeline for telecom domain."""

    def __init__(self, config: Dict[str, Any]):
        self.embeddings = OpenAIEmbeddings(
            model="text-embedding-3-large",
            dimensions=3072
        )
        self.llm = ChatOpenAI(
            model="gpt-4o-mini",
            temperature=0,
            max_tokens=2048,
            model_kwargs={"response_format": {"type": "json_object"}}
        )
        self.vector_store = self._setup_vector_store(config)
        self.qa_chain = self._create_qa_chain()

    def _setup_vector_store(self, config: Dict) -> Weaviate:
        """Initialize Weaviate vector database."""
        return Weaviate.from_existing_index(
            embedding=self.embeddings,
            index_name="telecom_knowledge",
            text_key="content",
            weaviate_url=config["weaviate_url"],
            additional_headers={"X-OpenAI-Api-Key": config["openai_api_key"]}
        )

    def _create_qa_chain(self) -> RetrievalQA:
        """Create telecom-optimized QA chain."""
        template = """
You are an expert telecom network engineer responsible for translating
natural language commands into structured JSON objects.
Your task is to analyze the user's command and determine if it is a
"NetworkFunctionDeployment" or a "NetworkFunctionScale" intent.

The user command is: "{question}"

Use the following context to inform your decision on default values if they
are not specified in the command.
Context: {context}

The output MUST be a single, valid JSON object. Do not add any explanation
or introductory text.

If the intent is to **deploy a new function**, use this JSON schema:
{{
  "type": "NetworkFunctionDeployment",
  "name": "string",
  "namespace": "string",
  "spec": {{ "replicas": "integer", "image": "string", "resources": {{...}} }},
  "o1_config": "string (XML)",
  "a1_policy": {{ "policy_type_id": "string", "policy_data": "object" }}
}}

If the intent is to **scale an existing function**, use this JSON schema:
{{
  "type": "NetworkFunctionScale",
  "name": "string (The name of the function to scale)",
  "namespace": "string",
  "replicas": "integer (The target number of replicas)"
}}

Based on the user command, generate the single, corresponding JSON object.
"""
        prompt = PromptTemplate(
            template=template,
            input_variables=["context", "question"]
        )

        return RetrievalQA.from_chain_type(
            llm=self.llm,
            chain_type="stuff",
            retriever=self.vector_store.as_retriever(
                search_type="mmr",
                search_kwargs={"k": 3}
            ),
            chain_type_kwargs={"prompt": prompt},
            return_source_documents=False
        )

    def process_intent(self, intent: str) -> Dict[str, Any]:
        """Process user intent and return a structured JSON object."""
        result = self.qa_chain.invoke({"query": intent})

        try:
            structured_output = json.loads(result['result'])
            structured_output['original_intent'] = intent
            return structured_output
        except (json.JSONDecodeError, TypeError) as e:
            return {
                "error": "Failed to parse LLM output as JSON.",
                "raw_output": result['result'],
                "original_intent": intent,
                "exception": str(e)
            }
