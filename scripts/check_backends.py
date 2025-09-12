#!/usr/bin/env python3
"""
Check and display status of all available storage backends for development.
"""

import os
import sys
import yaml
import sqlite3
import socket
from pathlib import Path
from typing import Dict, Any, Tuple
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from rich import box

console = Console()

def check_port(host: str, port: int, timeout: float = 1.0) -> bool:
    """Check if a port is open."""
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(timeout)
        result = sock.connect_ex((host, port))
        sock.close()
        return result == 0
    except:
        return False

def check_file_backend(config: Dict[str, Any]) -> Tuple[bool, str]:
    """Check if file backend is available."""
    path = Path(config.get('path', './data/file_storage'))
    if not path.exists():
        path.mkdir(parents=True, exist_ok=True)
        return True, f"Created: {path}"
    return True, f"Ready: {path}"

def check_sqlite_backend(config: Dict[str, Any]) -> Tuple[bool, str]:
    """Check if SQLite backend is available."""
    db_path = config.get('path', './data/pipeline_data.db')
    db_dir = Path(db_path).parent
    db_dir.mkdir(parents=True, exist_ok=True)
    
    try:
        conn = sqlite3.connect(db_path)
        conn.execute("SELECT 1")
        conn.close()
        return True, f"Ready: {db_path}"
    except Exception as e:
        return False, f"Error: {str(e)}"

def check_postgresql_backend(config: Dict[str, Any]) -> Tuple[bool, str]:
    """Check if PostgreSQL backend is available."""
    host = config.get('host', 'localhost')
    port = config.get('port', 5432)
    
    if check_port(host, port):
        return True, f"Running on {host}:{port}"
    else:
        docker_cmd = config.get('docker_command', '')
        return False, f"Not running. Start with:\n    {docker_cmd[:60]}..."

def check_mongodb_backend(config: Dict[str, Any]) -> Tuple[bool, str]:
    """Check if MongoDB backend is available."""
    host = config.get('host', 'localhost')
    port = config.get('port', 27017)
    
    if check_port(host, port):
        return True, f"Running on {host}:{port}"
    else:
        docker_cmd = config.get('docker_command', '')
        return False, f"Not running. Start with:\n    {docker_cmd[:60]}..."

def check_elasticsearch_backend(config: Dict[str, Any]) -> Tuple[bool, str]:
    """Check if Elasticsearch backend is available."""
    hosts = config.get('hosts', ['http://localhost:9200'])
    host = hosts[0].replace('http://', '').replace('https://', '').split(':')[0]
    port = int(hosts[0].split(':')[-1])
    
    if check_port(host, port):
        return True, f"Running on {hosts[0]}"
    else:
        docker_cmd = config.get('docker_command', '')
        return False, f"Not running. Start with:\n    {docker_cmd[:60]}..."

def check_neo4j_backend(config: Dict[str, Any]) -> Tuple[bool, str]:
    """Check if Neo4j backend is available."""
    uri = config.get('uri', 'bolt://localhost:7687')
    host = uri.replace('bolt://', '').replace('neo4j://', '').split(':')[0]
    port = int(uri.split(':')[-1])
    
    if check_port(host, port):
        return True, f"Running on {uri}"
    else:
        docker_cmd = config.get('docker_command', '')
        return False, f"Not running. Start with:\n    {docker_cmd[:60]}..."

def check_solr_backend(config: Dict[str, Any]) -> Tuple[bool, str]:
    """Check if Solr backend is available."""
    host = config.get('host', 'localhost')
    port = config.get('port', 8983)
    
    if check_port(host, port):
        return True, f"Running on {host}:{port}"
    else:
        docker_cmd = config.get('docker_command', '')
        return False, f"Not running. Start with:\n    {docker_cmd[:60]}..."

def check_sqlalchemy_backend(config: Dict[str, Any]) -> Tuple[bool, str]:
    """Check if SQLAlchemy backend is available."""
    conn_string = config.get('connection_string', '')
    if 'sqlite' in conn_string:
        return True, f"Ready: {conn_string}"
    else:
        return True, f"Configured: {conn_string[:40]}..."

def main():
    """Main function to check all backends."""
    # Load configuration
    config_path = Path(__file__).parent.parent / 'config' / 'dev-backends.yaml'
    
    if not config_path.exists():
        console.print("[red]Configuration file not found![/red]")
        console.print(f"Expected at: {config_path}")
        sys.exit(1)
    
    with open(config_path, 'r') as f:
        config = yaml.safe_load(f)
    
    # Print header
    console.print("\n")
    console.print(Panel.fit(
        "[bold cyan]Go-Doc-Go Development Environment[/bold cyan]\n"
        "[yellow]Storage Backend Status[/yellow]",
        box=box.DOUBLE
    ))
    
    # Create status table
    table = Table(
        title="Available Storage Backends",
        box=box.ROUNDED,
        show_header=True,
        header_style="bold magenta",
        expand=True
    )
    
    table.add_column("Backend", style="cyan", no_wrap=True)
    table.add_column("Status", no_wrap=True)
    table.add_column("Details")
    table.add_column("Description")
    
    # Backend checkers
    checkers = {
        'file': check_file_backend,
        'sqlite': check_sqlite_backend,
        'postgresql': check_postgresql_backend,
        'mongodb': check_mongodb_backend,
        'elasticsearch': check_elasticsearch_backend,
        'neo4j': check_neo4j_backend,
        'solr': check_solr_backend,
        'sqlalchemy': check_sqlalchemy_backend,
    }
    
    # Check each backend
    backends = config.get('storage_backends', {})
    available_count = 0
    
    for backend_name, backend_config in backends.items():
        if backend_config.get('enabled', False):
            checker = checkers.get(backend_name)
            if checker:
                is_available, details = checker(backend_config)
                status = "[green]✓ Ready[/green]" if is_available else "[red]✗ Not Available[/red]"
                if is_available:
                    available_count += 1
                
                description = backend_config.get('description', 'No description')
                table.add_row(
                    backend_name.upper(),
                    status,
                    details,
                    description
                )
    
    console.print(table)
    
    # Print summary
    total_backends = len(backends)
    console.print(f"\n[bold]Summary:[/bold] {available_count}/{total_backends} backends available")
    
    # Print quick start commands if any backends are unavailable
    if available_count < total_backends:
        console.print("\n[yellow]Quick Start Commands:[/yellow]")
        console.print("To start all backends with Docker Compose:")
        console.print("[cyan]docker-compose -f docker-compose.dev.yml up -d[/cyan]")
        console.print("\nOr start individual backends with the commands shown above.")
    
    # Print test content sources
    console.print("\n[bold]Test Content Sources:[/bold]")
    test_sources = config.get('test_content_sources', [])
    for source in test_sources:
        console.print(f"  • [cyan]{source['name']}[/cyan] ({source['type']}): {source.get('description', '')}")
    
    # Create test documents directory if it doesn't exist
    test_docs_path = Path('./test_documents')
    if not test_docs_path.exists():
        test_docs_path.mkdir(parents=True, exist_ok=True)
        console.print(f"\n[green]Created test documents directory:[/green] {test_docs_path}")
        
        # Create a sample file
        sample_file = test_docs_path / 'README.md'
        sample_file.write_text("""# Test Documents

This directory contains sample documents for testing the Go-Doc-Go pipeline.

## Sample Content

This is a test document that can be processed by the pipeline to verify that:
- Document parsing works correctly
- Content extraction functions properly  
- The storage backend saves data as expected
- Entity and relationship extraction operates correctly

### Test Entities
- **Organization**: Anthropic, OpenAI, Google
- **Person**: Claude, GPT, Gemini
- **Product**: Go-Doc-Go, Knowledge Engine
- **Location**: San Francisco, New York, London

### Test Relationships
- Claude WORKS_FOR Anthropic
- Go-Doc-Go IS_A Knowledge Engine
- Anthropic LOCATED_IN San Francisco
""")
        console.print(f"[green]Created sample document:[/green] {sample_file}")
    
    console.print("\n[bold green]Development environment ready![/bold green]")
    console.print("Frontend: http://localhost:5173")
    console.print("Backend API: http://localhost:8000")
    console.print("\n")

if __name__ == "__main__":
    # Check if rich is installed
    try:
        import rich
    except ImportError:
        print("Installing required package: rich")
        os.system(f"{sys.executable} -m pip install rich pyyaml")
    
    main()