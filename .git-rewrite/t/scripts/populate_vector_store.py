import os
import weaviate
from langchain_community.document_loaders import PyPDFLoader, TextLoader
from langchain.text_splitter import RecursiveCharacterTextSplitter
from langchain_openai import OpenAIEmbeddings
import logging

logging.basicConfig(level=logging.INFO)

# --- Configuration ---
KNOWLEDGE_BASE_DIR = "knowledge_base"
WEAVIATE_URL = os.environ.get("WEAVIATE_URL", "http://localhost:8080")
OPENAI_API_KEY = os.environ.get("OPENAI_API_KEY")
CLASS_NAME = "TelecomKnowledge"

def main():
    """
    Main function to ingest documents into the Weaviate vector store.
    """
    if not OPENAI_API_KEY:
        logging.error("OPENAI_API_KEY environment variable not set. Cannot proceed.")
        return

    # 1. Connect to Weaviate
    try:
        client = weaviate.Client(WEAVIATE_URL)
        logging.info(f"Successfully connected to Weaviate at {WEAVIATE_URL}")
    except Exception as e:
        logging.error(f"Failed to connect to Weaviate: {e}")
        return

    # 2. Define and create the Weaviate class schema
    class_obj = {
        "class": CLASS_NAME,
        "vectorizer": "none", # We will provide our own vectors
        "properties": [
            {
                "name": "content",
                "dataType": ["text"],
            },
            {
                "name": "source",
                "dataType": ["string"],
            }
        ]
    }
    if not client.schema.exists(CLASS_NAME):
        client.schema.create_class(class_obj)
        logging.info(f"Created Weaviate class: {CLASS_NAME}")

    # 3. Load documents from the knowledge base directory
    documents = []
    for root, _, files in os.walk(KNOWLEDGE_BASE_DIR):
        for file in files:
            file_path = os.path.join(root, file)
            try:
                if file.endswith(".pdf"):
                    loader = PyPDFLoader(file_path)
                elif file.endswith(".txt") or file.endswith(".md"):
                    loader = TextLoader(file_path)
                else:
                    continue
                documents.extend(loader.load())
                logging.info(f"Loaded {len(documents)} documents from {file_path}")
            except Exception as e:
                logging.error(f"Failed to load document {file_path}: {e}")

    if not documents:
        logging.warning("No documents found to ingest.")
        return

    # 4. Split documents into chunks
    text_splitter = RecursiveCharacterTextSplitter(chunk_size=1000, chunk_overlap=100)
    chunks = text_splitter.split_documents(documents)
    logging.info(f"Split {len(documents)} documents into {len(chunks)} chunks.")

    # 5. Generate embeddings and ingest into Weaviate
    embeddings = OpenAIEmbeddings(model="text-embedding-3-large", openai_api_key=OPENAI_API_KEY)
    
    client.batch.configure(batch_size=100)
    with client.batch as batch:
        for i, chunk in enumerate(chunks):
            vector = embeddings.embed_query(chunk.page_content)
            properties = {
                "content": chunk.page_content,
                "source": chunk.metadata.get("source", "Unknown"),
            }
            batch.add_data_object(
                properties,
                CLASS_NAME,
                vector=vector
            )
            logging.info(f"Added chunk {i+1}/{len(chunks)} to the batch.")

    logging.info("Successfully ingested all documents into Weaviate.")

if __name__ == "__main__":
    # As a simulation, we'll create a dummy knowledge base file.
    # In a real scenario, this directory would be populated with actual documentation.
    if not os.path.exists(KNOWLEDGE_BASE_DIR):
        os.makedirs(KNOWLEDGE_BASE_DIR)
    with open(os.path.join(KNOWLEDGE_BASE_DIR, "sample_spec.txt"), "w") as f:
        f.write("O-RAN Alliance specifications define open and intelligent RAN architectures.\n")
        f.write("Nephio R5 uses a Kubernetes-based control plane for intent-driven automation.\n")
    
    main()
