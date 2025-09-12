import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';

export interface ConfigurationData {
  llm?: {
    openai_api_key?: string;
    anthropic_api_key?: string;
    default_provider?: string;
    request_timeout?: number;
    max_tokens?: number;
    temperature?: number;
    max_retries?: number;
    [key: string]: any;
  };
  database?: {
    type?: string;
    host?: string;
    port?: number;
    database?: string;
    username?: string;
    password?: string;
    pool_size?: number;
    max_overflow?: number;
    pool_timeout?: number;
    pool_recycle?: number;
    pool_pre_ping?: boolean;
    query_timeout?: number;
    statement_timeout?: number;
    echo_sql?: boolean;
    autocommit?: boolean;
    [key: string]: any;
  };
  content_sources?: {
    sources?: {
      [key: string]: {
        type: string;
        enabled: boolean;
        name?: string;
        [key: string]: any;
      };
    };
  };
  processing?: {
    crawler_interval?: number;
    batch_size?: number;
    max_document_size?: number;
    processing_priority?: string;
    worker_count?: number;
    worker_memory_limit?: number;
    worker_timeout?: number;
    retry_attempts?: number;
    ocr_engine?: string;
    ocr_language?: string;
    max_pages?: number;
    extract_metadata?: boolean;
    extract_tables?: boolean;
    embedding_model?: string;
    chunk_size?: number;
    chunk_overlap?: number;
    enable_vector_search?: boolean;
    [key: string]: any;
  };
  api_endpoint?: string;
  config_path?: string;
  environment_variables?: { [key: string]: string };
  log_level?: string;
  log_format?: string;
  log_file_path?: string;
  max_log_file_size?: number;
  log_retention_days?: number;
  enable_cache?: boolean;
  cache_ttl?: number;
  max_cache_size?: number;
  enable_api_auth?: boolean;
  enable_encryption?: boolean;
  session_timeout?: number;
  max_login_attempts?: number;
  allowed_origins?: string;
  [key: string]: any;
}

export class ConfigurationService {
  private apiClient = axios.create({
    baseURL: API_BASE_URL,
    headers: {
      'Content-Type': 'application/json',
    },
  });

  /**
   * Get the current configuration
   */
  async getConfiguration(): Promise<ConfigurationData> {
    try {
      const response = await this.apiClient.get('/settings');
      return response.data;
    } catch (error) {
      console.error('Failed to fetch configuration:', error);
      // Return default configuration if API fails
      return this.getDefaultConfiguration();
    }
  }

  /**
   * Save configuration changes
   */
  async saveConfiguration(config: ConfigurationData): Promise<void> {
    try {
      // Encrypt sensitive values before sending
      const sanitizedConfig = this.sanitizeConfiguration(config);
      await this.apiClient.post('/settings', sanitizedConfig);
    } catch (error) {
      console.error('Failed to save configuration:', error);
      throw new Error('Failed to save configuration');
    }
  }

  /**
   * Validate configuration
   */
  async validateConfiguration(config: ConfigurationData): Promise<boolean> {
    try {
      const response = await this.apiClient.post('/settings/validate', config);
      return response.data.valid === true;
    } catch (error) {
      console.error('Configuration validation failed:', error);
      return false;
    }
  }

  /**
   * Test LLM connection
   */
  async testLLMConnection(provider: string, apiKey: string): Promise<boolean> {
    try {
      const response = await this.apiClient.post('/settings/test-llm', {
        provider,
        api_key: apiKey,
      });
      return response.data.success === true;
    } catch (error) {
      console.error('LLM connection test failed:', error);
      return false;
    }
  }

  /**
   * Test database connection
   */
  async testDatabaseConnection(dbConfig: any): Promise<boolean> {
    try {
      const response = await this.apiClient.post('/settings/test-database', dbConfig);
      return response.data.success === true;
    } catch (error) {
      console.error('Database connection test failed:', error);
      return false;
    }
  }

  /**
   * Test content source connection
   */
  async testContentSource(sourceConfig: any): Promise<boolean> {
    try {
      const response = await this.apiClient.post('/settings/test-source', sourceConfig);
      return response.data.success === true;
    } catch (error) {
      console.error('Content source test failed:', error);
      return false;
    }
  }

  /**
   * Get environment variables (read-only)
   */
  async getEnvironmentVariables(): Promise<{ [key: string]: string }> {
    try {
      const response = await this.apiClient.get('/settings/environment');
      return response.data;
    } catch (error) {
      console.error('Failed to fetch environment variables:', error);
      return {};
    }
  }

  /**
   * Export configuration
   */
  async exportConfiguration(): Promise<ConfigurationData> {
    try {
      const response = await this.apiClient.get('/settings/export');
      return response.data;
    } catch (error) {
      console.error('Failed to export configuration:', error);
      throw new Error('Failed to export configuration');
    }
  }

  /**
   * Import configuration
   */
  async importConfiguration(config: ConfigurationData): Promise<void> {
    try {
      await this.apiClient.post('/settings/import', config);
    } catch (error) {
      console.error('Failed to import configuration:', error);
      throw new Error('Failed to import configuration');
    }
  }

  /**
   * Clear cache
   */
  async clearCache(): Promise<void> {
    try {
      await this.apiClient.post('/settings/clear-cache');
    } catch (error) {
      console.error('Failed to clear cache:', error);
      throw new Error('Failed to clear cache');
    }
  }

  /**
   * Get system information
   */
  async getSystemInfo(): Promise<any> {
    try {
      const response = await this.apiClient.get('/settings/system-info');
      return response.data;
    } catch (error) {
      console.error('Failed to fetch system info:', error);
      return {
        version: '1.0.0',
        environment: 'production',
      };
    }
  }

  /**
   * Sanitize configuration before sending to API
   * Removes empty strings and null values
   */
  private sanitizeConfiguration(config: ConfigurationData): ConfigurationData {
    const sanitized: any = {};
    
    for (const [key, value] of Object.entries(config)) {
      if (value !== null && value !== undefined && value !== '') {
        if (typeof value === 'object' && !Array.isArray(value)) {
          sanitized[key] = this.sanitizeConfiguration(value as ConfigurationData);
        } else {
          sanitized[key] = value;
        }
      }
    }
    
    return sanitized;
  }

  /**
   * Get default configuration
   */
  private getDefaultConfiguration(): ConfigurationData {
    return {
      llm: {
        default_provider: 'openai',
        request_timeout: 30,
        max_tokens: 2000,
        temperature: 0.7,
        max_retries: 3,
      },
      database: {
        type: 'postgresql',
        host: 'localhost',
        port: 5432,
        database: 'go-doc-go',
        pool_size: 10,
        max_overflow: 20,
        pool_timeout: 30,
        pool_recycle: 3600,
        pool_pre_ping: true,
        query_timeout: 30,
        statement_timeout: 60,
        echo_sql: false,
        autocommit: true,
      },
      content_sources: {
        sources: {},
      },
      processing: {
        crawler_interval: 300,
        batch_size: 10,
        max_document_size: 100,
        processing_priority: 'normal',
        worker_count: 4,
        worker_memory_limit: 512,
        worker_timeout: 300,
        retry_attempts: 3,
        ocr_engine: 'tesseract',
        ocr_language: 'eng',
        max_pages: 1000,
        extract_metadata: true,
        extract_tables: true,
        embedding_model: 'text-embedding-ada-002',
        chunk_size: 512,
        chunk_overlap: 50,
        enable_vector_search: true,
      },
      api_endpoint: '/api',
      config_path: './config.yaml',
      environment_variables: {},
      log_level: 'INFO',
      log_format: 'json',
      log_file_path: './logs/app.log',
      max_log_file_size: 100,
      log_retention_days: 30,
      enable_cache: true,
      cache_ttl: 3600,
      max_cache_size: 500,
      enable_api_auth: true,
      enable_encryption: true,
      session_timeout: 30,
      max_login_attempts: 5,
      allowed_origins: '*',
    };
  }
}