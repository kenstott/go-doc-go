import axios, { AxiosInstance, AxiosError } from 'axios';
import { Config, Ontology, Domain, OntologyListItem, ApiResponse } from '../types';

// Create axios instance with default config
const api: AxiosInstance = axios.create({
  baseURL: '/api',  // Use relative URL to go through Vite proxy
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor
api.interceptors.request.use(
  (config) => {
    // Add auth token if available
    const token = localStorage.getItem('auth_token');
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    
    // Debug logging for pipeline updates
    if (config.url?.includes('/pipelines/') && config.method === 'put') {
      console.log('=== AXIOS INTERCEPTOR DEBUG ===');
      console.log('URL:', config.url);
      console.log('Method:', config.method);
      console.log('Data being sent:', config.data);
      if (typeof config.data === 'string') {
        try {
          const parsed = JSON.parse(config.data);
          console.log('Parsed data:', parsed);
          console.log('expected_version in parsed data:', parsed.expected_version);
        } catch (e) {
          console.log('Could not parse data as JSON');
        }
      }
    }
    
    return config;
  },
  (error: AxiosError) => {
    return Promise.reject(error);
  }
);

// Response interceptor
api.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      // Handle unauthorized
      console.error('Unauthorized access');
    }
    return Promise.reject(error);
  }
);

// Configuration endpoints
export const configApi = {
  get: async (): Promise<Config> => {
    const response = await api.get<Config>('/config');
    return response.data;
  },
  
  update: async (config: Config): Promise<ApiResponse> => {
    const response = await api.post<ApiResponse>('/config', config);
    return response.data;
  },
  
  validate: async (config: Config): Promise<ApiResponse> => {
    const response = await api.post<ApiResponse>('/config/validate', config);
    return response.data;
  },
};

// Ontology endpoints
export const ontologyApi = {
  listOntologies: async () => {
    const response = await api.get('/ontologies');
    return response.data;
  },
  
  get: async (name: string): Promise<Ontology> => {
    const response = await api.get<Ontology>(`/ontologies/${name}`);
    return response.data;
  },
  
  createOntology: async (name: string, ontology: Ontology): Promise<ApiResponse> => {
    const response = await api.post<ApiResponse>(`/ontologies/${name}`, ontology);
    return response.data;
  },
  
  updateOntology: async (name: string, ontology: Ontology): Promise<ApiResponse> => {
    const response = await api.put<ApiResponse>(`/ontologies/${name}`, ontology);
    return response.data;
  },
  
  validateOntology: async (ontology: Ontology): Promise<ApiResponse> => {
    const response = await api.post<ApiResponse>('/ontologies/validate', ontology);
    return response.data;
  },
  
  delete: async (name: string): Promise<ApiResponse> => {
    const response = await api.delete<ApiResponse>(`/ontologies/${name}`);
    return response.data;
  },
};

// Domain endpoints
export const domainApi = {
  listDomains: async () => {
    const response = await api.get('/domain/active');
    return response.data;
  },
  
  activateDomain: async (name: string): Promise<ApiResponse> => {
    const response = await api.post<ApiResponse>(`/domain/${name}/activate`);
    return response.data;
  },
  
  deactivateDomain: async (name: string): Promise<ApiResponse> => {
    const response = await api.post<ApiResponse>(`/domain/${name}/deactivate`);
    return response.data;
  },
};

// Pipeline endpoints
export const pipelineApi = {
  list: async () => {
    const response = await api.get('/pipelines');
    return response.data;
  },
  
  get: async (id: number) => {
    const response = await api.get(`/pipelines/${id}`);
    return response.data;
  },
  
  create: async (pipeline: {
    name: string;
    description: string;
    config_yaml: string;
    tags?: string[];
  }) => {
    const response = await api.post('/pipelines', pipeline);
    return response.data;
  },
  
  update: async (id: number, pipeline: {
    name?: string;
    description?: string;
    config_yaml?: string;
    tags?: string[];
    is_active?: boolean;
    expected_version: number;  // Required for optimistic locking
  }) => {
    console.log('=== API.TS UPDATE DEBUG ===');
    console.log('1. Full pipeline object received:', pipeline);
    console.log('2. expected_version value:', pipeline.expected_version);
    console.log('3. expected_version type:', typeof pipeline.expected_version);
    console.log('4. Keys in pipeline object:', Object.keys(pipeline));
    
    // Ensure expected_version is included in the request
    const requestPayload = {
      ...pipeline,
      expected_version: pipeline.expected_version
    };
    
    console.log('5. Request payload being sent:', JSON.stringify(requestPayload, null, 2));
    
    const response = await api.put(`/pipelines/${id}`, requestPayload);
    console.log('6. Response from server:', response.data);
    return response.data;
  },
  
  delete: async (id: number) => {
    const response = await api.delete(`/pipelines/${id}`);
    return response.data;
  },
  
  execute: async (id: number, params?: {
    worker_count?: number;
    documents_total?: number;
  }) => {
    const response = await api.post(`/pipelines/${id}/execute`, params || {});
    return response.data;
  },
  
  getExecutions: async (pipelineId?: number) => {
    const url = pipelineId ? `/pipelines/${pipelineId}/executions` : '/pipelines/executions';
    const response = await api.get(url);
    return response.data;
  },
  
  getExecution: async (runId: string) => {
    const response = await api.get(`/pipelines/executions/${runId}`);
    return response.data;
  },
  
  cancelExecution: async (runId: string) => {
    const response = await api.post(`/pipelines/executions/${runId}/cancel`);
    return response.data;
  },
  
  getActiveExecutions: async () => {
    const response = await api.get('/pipelines/executions/active');
    return response.data;
  },

  getRecentExecutions: async (limit: number = 20) => {
    const response = await api.get(`/pipelines/executions/recent?limit=${limit}`);
    return response.data;
  },
  
  getTemplates: async () => {
    const response = await api.get('/pipelines/templates');
    return response.data;
  },
  
  createFromTemplate: async (templateId: number, name: string, description?: string) => {
    const response = await api.post('/pipelines/from-template', {
      template_id: templateId,
      name,
      description
    });
    return response.data;
  },
  
  clone: async (id: number, newName: string) => {
    const response = await api.post(`/pipelines/${id}/clone`, { name: newName });
    return response.data;
  },
  
  export: async (id: number) => {
    const response = await api.get(`/pipelines/${id}/export`, {
      responseType: 'blob'
    });
    return response.data;
  },
  
  import: async (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    const response = await api.post('/pipelines/import', formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    });
    return response.data;
  },

  query: async (pipelineId: number, queryOptions: {
    query: string;
    limit?: number;
    offset?: number;
    similarity_threshold?: number;
    filters?: {
      date_range?: {
        operator: string;
        relative_value?: number;
        start_date?: string;
        end_date?: string;
        date?: string;
      };
      element_types?: string[];
      document_types?: string[];
      metadata?: Record<string, any>;
    };
    include_content?: boolean;
    include_metadata?: boolean;
  }) => {
    const response = await api.post(`/pipelines/${pipelineId}/query`, queryOptions);
    return response.data;
  },

  getElementContent: async (pipelineId: number, elementId: string) => {
    // Direct lookup using the dedicated endpoint
    const response = await api.get(`/pipelines/${pipelineId}/elements/${elementId}/content`);
    return response.data;
  }
};

export default api;