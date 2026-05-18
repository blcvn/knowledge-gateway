import { apiClient } from '../lib/api-client';
import { API_CONFIG } from '../config/api.config';
import type { SubgraphData, OntologySchema } from '../types/graph';

const BASE = API_CONFIG.console.graph;

export const graphService = {
  /** POST /v1/console/graph/subgraph */
  getSubgraph: (params: Record<string, string>) =>
    apiClient.post<SubgraphData>(`${BASE}/subgraph`, params),

  /** GET /v1/console/graph/entity/{id} */
  getEntity: (id: string) =>
    apiClient.get<Record<string, unknown>>(`${BASE}/entity/${id}`),

  /** POST /v1/console/graph/timeline */
  getTimeline: (params: Record<string, string>) =>
    apiClient.post<unknown[]>(`${BASE}/timeline`, params),

  /** GET /v1/console/graph/ontology */
  getOntology: () =>
    apiClient.get<OntologySchema>(`${BASE}/ontology`),

  /** PUT /v1/console/graph/ontology */
  updateOntology: (schema: OntologySchema) =>
    apiClient.put<OntologySchema>(`${BASE}/ontology`, schema),

  /** POST /v1/console/graph/query */
  query: (cypher: string) =>
    apiClient.post<SubgraphData>(`${BASE}/query`, { query: cypher }),
};
