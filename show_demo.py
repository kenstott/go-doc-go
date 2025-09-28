#!/usr/bin/env python3
"""
Show the demo without interactive input - for displaying the complete flow
"""

import time
from demo_discovery_interview import DemoDiscoveryInterview

# Create demo instance
demo = DemoDiscoveryInterview("/Volumes/T9/sec_analytics", "financial")

# Import Colors class
from demo_discovery_interview import Colors

# Override the wait_for_enter method to auto-proceed
def auto_continue(prompt="Press Enter to continue..."):
    print(f"\n{Colors.YELLOW}👆 {prompt}{Colors.END}")
    print(f"{Colors.CYAN}[Auto-continuing in demo mode...]{Colors.END}")
    time.sleep(1)  # Brief pause to show what's happening

demo._wait_for_enter = auto_continue

# Override simulate_processing for faster demo
def fast_simulate(message, duration=2.0):
    print(f"{Colors.BLUE}{message}", end="", flush=True)
    for i in range(6):  # Fixed number of dots
        print(".", end="", flush=True)
        time.sleep(0.2)
    print(f" Done!{Colors.END}")

demo._simulate_processing = fast_simulate

# Run the demo
print("🎬 SHOWING GO-DOC-GO DISCOVERY DEMO")
print("=" * 50)
demo.run_demo()