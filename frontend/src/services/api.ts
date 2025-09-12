import axios, { AxiosInstance, AxiosError } from 'axios';
import { Config, Ontology, Domain, OntologyListItem, ApiResponse } from '../types';

// Create axios instance with default config
const api: AxiosInstance = axios.create({
  baseURL: 'http://localhost:5002/api',
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

export default api;