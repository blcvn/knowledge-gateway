import os
import requests
from dotenv import load_dotenv

def get_config():
    # Load .env from tests/client/.env first
    local_env = os.path.join(os.path.dirname(__file__), '.env')
    if os.path.exists(local_env):
        load_dotenv(local_env)
    
    # Fallback to deploy/dev/.env if missing
    dev_env = os.path.join(os.path.dirname(__file__), '../../deploy/dev/.env')
    if os.path.exists(dev_env):
        load_dotenv(dev_env)
    
    host = os.environ.get("KG_HTTP_HOST", "172.20.2.39")
    if host == "0.0.0.0":
        host = "172.20.2.39"
    port = os.environ.get("KG_HTTP_PORT", "18082")
    api_key = os.environ.get("KG_API_KEY", "kgsk_test_alpha_admin")
    
    base_url = f"http://{host}:{port}"
    return base_url, api_key

def test_api():
    base_url, api_key = get_config()
    print(f"Testing API against {base_url}...")
    
    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json"
    }
    
    # 1. Health check
    print("\n1. Testing GET /healthz")
    resp = requests.get(f"{base_url}/healthz")
    print(f"Status: {resp.status_code}")
    print(resp.json())
    
    # 2. Access resolve
    print("\n2. Testing GET /v1/access/resolve")
    resp = requests.get(f"{base_url}/v1/access/resolve", headers=headers)
    print(f"Status: {resp.status_code}")
    if resp.status_code == 200:
        print(resp.json())
    else:
        print("Failed to resolve access. Ensure the API key is correct and database is seeded.")
        return
        
    # 3. Read templates for sample-policy
    print("\n3. Testing GET /v1/kg/read/templates?domain_id=sample-policy")
    resp = requests.get(f"{base_url}/v1/kg/read/templates?domain_id=sample-policy", headers=headers)
    print(f"Status: {resp.status_code}")
    if resp.status_code == 200:
        print(resp.json())
        
    # 4. Search semantic (if vector adapter works)
    print("\n4. Testing POST /v1/kg/search/semantic")
    payload = {
        "query": "policy routing",
        "domain_ids": ["sample-policy"],
        "top_k": 3
    }
    resp = requests.post(f"{base_url}/v1/kg/search/semantic", headers=headers, json=payload)
    print(f"Status: {resp.status_code}")
    if resp.status_code == 200:
        print("Semantic search successful. Found results:", len(resp.json().get('data', [])))
    else:
        print(resp.text)

if __name__ == "__main__":
    test_api()
