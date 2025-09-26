"""
Test analytics CLI functionality.
"""

import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import MagicMock, patch, Mock
from click.testing import CliRunner

from go_doc_go.cli.analytics import main, DatabaseAgnosticAnalyticsManager
from go_doc_go.config import Config


class TestAnalyticsCLI(unittest.TestCase):
    """Test analytics CLI functionality."""

    def setUp(self):
        """Set up test environment."""
        self.test_dir = tempfile.mkdtemp()
        self.config_path = Path(self.test_dir) / "config.yaml"
        self.analytics_path = Path(self.test_dir) / "analytics-output"
        self.analytics_path.mkdir(exist_ok=True)

        # Create test configuration
        config_data = {
            'analytics': {
                'enabled': True,
                'outputs': [
                    {
                        'type': 'parquet',
                        'path': str(self.analytics_path),
                        'partitioning': {
                            'scheme': 'date_source',
                            'compression': 'snappy'
                        }
                    }
                ]
            }
        }

        import yaml
        with open(self.config_path, 'w') as f:
            yaml.dump(config_data, f)

        # Create some mock parquet files for testing
        self._create_mock_analytics_data()

    def _create_mock_analytics_data(self):
        """Create mock analytics data structure."""
        # Create directory structure that analytics would create
        tables = ['documents', 'elements', 'relationships', 'embeddings']
        for table in tables:
            table_dir = self.analytics_path / table / "date=2025-01-01" / "source=test"
            table_dir.mkdir(parents=True, exist_ok=True)
            # Create empty parquet file
            (table_dir / f"{table}_0.parquet").touch()

    def tearDown(self):
        """Clean up test environment."""
        import shutil
        if os.path.exists(self.test_dir):
            shutil.rmtree(self.test_dir)

    def test_analytics_cli_basic(self):
        """Test basic analytics CLI functionality."""
        runner = CliRunner()
        result = runner.invoke(main, ['--config', str(self.config_path)])

        # Check that command runs without error
        self.assertEqual(result.exit_code, 0, f"CLI failed: {result.output}")

        # Check for expected output
        self.assertIn("GO-DOC-GO ANALYTICS SUMMARY", result.output)
        self.assertIn("Analytics Enabled:   ✅", result.output)
        self.assertIn("Storage Backend:", result.output)

    def test_analytics_cli_detailed(self):
        """Test analytics CLI with detailed flag."""
        runner = CliRunner()
        result = runner.invoke(main, ['--config', str(self.config_path), '--detailed'])

        self.assertEqual(result.exit_code, 0, f"CLI failed: {result.output}")

        # Detailed view should show more information
        self.assertIn("GO-DOC-GO ANALYTICS SUMMARY", result.output)

    def test_analytics_cli_json_output(self):
        """Test analytics CLI with JSON output."""
        runner = CliRunner()
        result = runner.invoke(main, ['--config', str(self.config_path), '--json'])

        self.assertEqual(result.exit_code, 0, f"CLI failed: {result.output}")

        # Should produce valid JSON
        try:
            data = json.loads(result.output)
            self.assertIn('timestamp', data)
            self.assertIn('analytics_enabled', data)
            self.assertTrue(data['analytics_enabled'])
        except json.JSONDecodeError:
            self.fail(f"Invalid JSON output: {result.output}")

    def test_analytics_cli_disabled(self):
        """Test analytics CLI when analytics is disabled."""
        # Create config with analytics disabled
        config_data = {
            'analytics': {
                'enabled': False,
                'outputs': []
            }
        }

        import yaml
        disabled_config_path = Path(self.test_dir) / "disabled_config.yaml"
        with open(disabled_config_path, 'w') as f:
            yaml.dump(config_data, f)

        runner = CliRunner()
        result = runner.invoke(main, ['--config', str(disabled_config_path)])

        # Should exit with error
        self.assertNotEqual(result.exit_code, 0)
        self.assertIn("Analytics is not enabled", result.output)

    def test_analytics_cli_missing_config(self):
        """Test analytics CLI with missing config file."""
        runner = CliRunner()
        result = runner.invoke(main, ['--config', '/nonexistent/config.yaml'])

        # Should exit with error
        self.assertNotEqual(result.exit_code, 0)
        self.assertIn("Configuration file not found", result.output)

    def test_analytics_manager_initialization(self):
        """Test DatabaseAgnosticAnalyticsManager initialization."""
        config = Config(str(self.config_path))

        # Should initialize without error
        manager = DatabaseAgnosticAnalyticsManager(config)
        self.assertIsNotNone(manager.analytics_storage)
        self.assertIsNotNone(manager.config)

    def test_analytics_manager_no_outputs(self):
        """Test manager initialization with no analytics outputs."""
        # Create config with no outputs
        config_data = {
            'analytics': {
                'enabled': True,
                'outputs': []
            }
        }

        import yaml
        no_outputs_config = Path(self.test_dir) / "no_outputs_config.yaml"
        with open(no_outputs_config, 'w') as f:
            yaml.dump(config_data, f)

        config = Config(str(no_outputs_config))

        # Should raise ValueError
        with self.assertRaises(ValueError) as context:
            DatabaseAgnosticAnalyticsManager(config)
        self.assertIn("No analytics storage configured", str(context.exception))

    @patch('go_doc_go.storage_adapters.factory.StorageFactory.create_analytics_storage')
    def test_analytics_summary_generation(self, mock_create_storage):
        """Test analytics summary generation."""
        # Mock the analytics storage
        mock_storage = Mock()
        mock_storage.get_storage_summary.return_value = {
            'backend': 'parquet',
            'path': str(self.analytics_path),
            'storage_health': 'healthy',
            'total_size': 1024 * 1024,
            'table_counts': {
                'documents': 100,
                'elements': 500,
                'relationships': 200,
                'embeddings': 100
            }
        }
        mock_storage.get_table_statistics.return_value = {
            'tables': [
                {
                    'name': 'documents',
                    'type': 'parquet',
                    'row_count': 100,
                    'total_size': 512 * 1024
                }
            ]
        }
        mock_storage.get_run_statistics.return_value = {
            'total_runs': 5,
            'runs': [
                {
                    'run_id': 'test-run-1',
                    'start_time': '2025-01-01T00:00:00',
                    'document_count': 50
                }
            ]
        }
        mock_create_storage.return_value = mock_storage

        config = Config(str(self.config_path))
        manager = DatabaseAgnosticAnalyticsManager(config)

        summary = manager.get_analytics_summary()

        # Verify summary structure
        self.assertIn('timestamp', summary)
        self.assertIn('analytics_enabled', summary)
        self.assertTrue(summary['analytics_enabled'])
        self.assertEqual(summary['backend'], 'parquet')
        self.assertEqual(summary['storage_health'], 'healthy')
        self.assertEqual(summary['table_counts']['documents'], 100)
        self.assertEqual(summary['total_runs'], 5)
        self.assertEqual(len(summary['recent_runs']), 1)

    @patch('go_doc_go.storage_adapters.factory.StorageFactory.create_analytics_storage')
    def test_analytics_summary_with_error(self, mock_create_storage):
        """Test analytics summary generation when storage raises error."""
        # Mock storage that raises an error
        mock_storage = Mock()
        mock_storage.get_storage_summary.side_effect = Exception("Storage error")
        mock_create_storage.return_value = mock_storage

        config = Config(str(self.config_path))
        manager = DatabaseAgnosticAnalyticsManager(config)

        summary = manager.get_analytics_summary()

        # Should capture error in summary
        self.assertIn('error', summary)
        self.assertEqual(summary['error'], "Storage error")
        self.assertEqual(summary['storage_health'], 'error')

    def test_analytics_cli_with_environment_variable(self):
        """Test analytics CLI using environment variable for config."""
        # Set environment variable
        os.environ['GO_DOC_GO_CONFIG_PATH'] = str(self.config_path)

        try:
            runner = CliRunner()
            # Don't specify --config, should use env var
            result = runner.invoke(main, [])

            self.assertEqual(result.exit_code, 0, f"CLI failed: {result.output}")
            self.assertIn("GO-DOC-GO ANALYTICS SUMMARY", result.output)
        finally:
            # Clean up environment variable
            if 'GO_DOC_GO_CONFIG_PATH' in os.environ:
                del os.environ['GO_DOC_GO_CONFIG_PATH']

    def test_format_helpers(self):
        """Test formatting helper functions."""
        from go_doc_go.cli.analytics import format_timestamp, format_file_size

        # Test timestamp formatting
        self.assertEqual(format_timestamp(None), "N/A")
        self.assertEqual(format_timestamp(""), "N/A")

        # ISO format timestamp
        iso_time = "2025-01-01T12:30:45Z"
        formatted = format_timestamp(iso_time)
        self.assertIn("2025-01-01", formatted)
        self.assertIn("12:30:45", formatted)

        # Test file size formatting
        self.assertEqual(format_file_size(None), "Unknown")
        self.assertEqual(format_file_size(100), "100.0 B")
        self.assertEqual(format_file_size(1024), "1.0 KB")
        self.assertEqual(format_file_size(1024 * 1024), "1.0 MB")
        self.assertEqual(format_file_size(1024 * 1024 * 1024), "1.0 GB")


if __name__ == '__main__':
    unittest.main()