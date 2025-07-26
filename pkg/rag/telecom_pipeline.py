from typing import List, Dict, Any
from langchain_community.vectorstores import Weaviate
from langchain_openai import ChatOpenAI, OpenAIEmbeddings
from langchain.chains import RetrievalQA
from langchain.prompts import PromptTemplate

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
            max_tokens=2048
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
        prompt = PromptTemplate(
            template="""You are an expert telecom network engineer. Use the context to provide accurate technical answers.

Context: {context}

Question: {question}

Provide a technical answer including:
- Relevant standards (3GPP, O-RAN, ITU)
- Implementation details
- Configuration examples where applicable
- Network function interactions

Answer:""",
            input_variables=["context", "question"]
        )
        
        return RetrievalQA.from_chain_type(
            llm=self.llm,
            chain_type="stuff",
            retriever=self.vector_store.as_retriever(
                search_type="mmr",
                search_kwargs={"k": 6, "fetch_k": 20}
            ),
            chain_type_kwargs={"prompt": prompt},
            return_source_documents=True
        )
    
    def _calculate_confidence(self, result: Dict[str, Any]) -> float:
        """Calculate confidence score based on the retrieval results."""
        # This is a placeholder implementation. A more sophisticated approach
        # would analyze the scores of the retrieved documents.
        if result and result.get("source_documents"):
            return 1.0 
        return 0.0

    def process_intent(self, intent: str) -> Dict[str, Any]:
        """Process natural language intent and return structured output."""
        result = self.qa_chain({"query": intent})
        
        return {
            "intent": intent,
            "interpretation": result["result"],
            "confidence": self._calculate_confidence(result),
            "sources": [
                {
                    "content": doc.page_content[:200],
                    "metadata": doc.metadata
                }
                for doc in result["source_documents"]
            ]
        }
