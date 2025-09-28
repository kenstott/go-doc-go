#!/usr/bin/env python3
"""
CLI for Demo Ontology Discovery
"""

import click
import sys
from pathlib import Path

# Add project root to path for demo modules
project_root = Path(__file__).parent.parent.parent.parent
sys.path.insert(0, str(project_root))


@click.command()
@click.option('--analytics', '-a', default="/Volumes/T9/sec_analytics",
              help='Path to analytics parquet files')
@click.option('--mode', type=click.Choice(['interactive', 'auto']), default='interactive',
              help='Demo mode: interactive (press Enter) or auto (10s delays)')
@click.option('--fast', '-f', is_flag=True,
              help='Fast mode - shorter delays between steps')
@click.option('--neo4j', is_flag=True,
              help='Setup Neo4j and import results at end of demo')
def main(analytics, mode, fast, neo4j):
    """
    Run a live demo of the ontology discovery process.

    This demo automatically walks through the complete discovery workflow
    using REAL analytics data. The domain is auto-discovered from the data.
    You control the pacing by pressing Enter at each step.

    With --neo4j flag, the demo will setup Neo4j, import the results,
    and launch Bloom for hands-on exploration.

    Perfect for demonstrations and showcasing Go-Doc-Go discovery capabilities.
    """

    try:
        from demo_discovery_interview import DemoDiscoveryInterview

        click.echo("🎬 Starting Go-Doc-Go Discovery Demo...")

        # Create demo instance (domain will be auto-discovered)
        demo = DemoDiscoveryInterview(analytics_path=analytics)

        # Configure Neo4j integration
        demo.enable_neo4j = neo4j

        # Configure demo mode
        if mode == 'auto':
            # Auto mode - wait 10 seconds (or 3 seconds if fast) between steps
            def auto_wait(prompt="Press Enter to continue..."):
                wait_time = 3 if fast else 10
                click.echo(f"\n🕐 {prompt} [Auto-continuing in {wait_time} seconds...]")
                import time
                time.sleep(wait_time)
            demo._wait_for_enter = auto_wait

        # Adjust timing for fast mode
        if fast:
            # Monkey patch the simulate_processing method for faster demo
            original_simulate = demo._simulate_processing
            def fast_simulate(message, duration=2.0):
                return original_simulate(message, min(duration, 0.5))
            demo._simulate_processing = fast_simulate

        # Run the demo
        demo.run_demo()

    except ImportError as e:
        click.echo(f"❌ Demo module not found: {e}", err=True)
        click.echo("Make sure demo_discovery_interview.py is in the project root", err=True)
        sys.exit(1)

    except KeyboardInterrupt:
        click.echo(f"\n\n🛑 Demo interrupted by user")
        sys.exit(0)

    except Exception as e:
        click.echo(f"❌ Demo failed: {e}", err=True)
        import traceback
        if click.get_current_context().obj and click.get_current_context().obj.get('verbose'):
            traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()