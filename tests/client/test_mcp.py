import os
import json
import requests
import sseclient
import threading
import time
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

def test_mcp():
    base_url, api_key = get_config()
    print(f"Testing MCP against {base_url}...")
    
    headers = {
        "Authorization": f"Bearer {api_key}"
    }
    
    # 1. Connect and get Session ID
    print("\n1. Connecting to SSE endpoint /v1/mcp/connect")
    
    # We run the SSE client in a separate thread so it doesn't block our testing thread.
    session_id = None
    connected_event = threading.Event()
    
    def listen_sse():
        nonlocal session_id
        response = requests.get(f"{base_url}/v1/mcp/connect", headers=headers, stream=True)
        if response.status_code != 200:
            print(f"Failed to connect: {response.status_code} {response.text}")
            return
            
        client = sseclient.SSEClient(response)
        for event in client.events():
            if event.event == 'session':
                try:
                    data = json.loads(event.data)
                    session_id = data.get("session_id")
                    print(f"Received session_id: {session_id}")
                    connected_event.set()
                except Exception as e:
                    print("Error parsing session event:", e)
            else:
                print(f"Received event: {event.event} -> {event.data}")
                
    sse_thread = threading.Thread(target=listen_sse, daemon=True)
    sse_thread.start()
    
    # Wait for the session event
    if not connected_event.wait(timeout=5.0):
        print("Timeout waiting for MCP session.")
        return
        
    rpc_headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json"
    }
    
    # 2. List tools
    print("\n2. Calling JSON-RPC: tools/list")
    payload_list = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/list"
    }
    resp = requests.post(f"{base_url}/v1/mcp/messages/{session_id}", headers=rpc_headers, json=payload_list)
    print(f"Status: {resp.status_code}")
    print(json.dumps(resp.json(), indent=2))
    
    # 3. Call a tool (kg_list_templates)
    print("\n3. Calling JSON-RPC: tools/call (kg_list_templates)")
    payload_call = {
        "jsonrpc": "2.0",
        "id": 2,
        "method": "tools/call",
        "params": {
            "name": "kg_list_templates",
            "arguments": {
                "domain_id": "sample-policy"
            }
        }
    }
    resp = requests.post(f"{base_url}/v1/mcp/messages/{session_id}", headers=rpc_headers, json=payload_call)
    print(f"Status: {resp.status_code}")
    print(json.dumps(resp.json(), indent=2))
    
    print("\nTests completed.")

if __name__ == "__main__":
    test_mcp()
