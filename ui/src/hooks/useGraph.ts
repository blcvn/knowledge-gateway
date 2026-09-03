/**
 * Graph Hooks — real API, no mock
 * TASK-API-014: Remove final mock dependency
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { graphService } from '../services/graph.service';
import type { OntologySchema, SubgraphData } from '../types/graph';

// ─── Query Key Factory ────────────────────────────────────────────────────────

const keys = {
  subgraph:  (p: Record<string, string>) => ['graph', 'subgraph', p] as const,
  timeline:  (p: Record<string, string>) => ['graph', 'timeline', p] as const,
  entity:    (id: string) => ['graph', 'entity', id] as const,
  ontology:  () => ['graph', 'ontology'] as const,
};

// ─── Query Hooks ──────────────────────────────────────────────────────────────

/** POST /v1/console/graph/subgraph */
export function useSubgraph(params: Record<string, string>) {
  return useQuery({
    queryKey: keys.subgraph(params),
    queryFn:  () => graphService.getSubgraph(params),
  });
}

/** POST /v1/console/graph/timeline */
export function useTimeline(params: Record<string, string>) {
  return useQuery({
    queryKey: keys.timeline(params),
    queryFn:  () => graphService.getTimeline(params),
  });
}

/** GET /v1/console/graph/entity/{id} */
export function useEntityDetail(id: string) {
  return useQuery({
    queryKey: keys.entity(id),
    queryFn:  () => graphService.getEntity(id),
    enabled:  !!id,
  });
}

/** GET /v1/console/graph/ontology */
export function useOntology() {
  return useQuery({
    queryKey: keys.ontology(),
    queryFn:  () => graphService.getOntology(),
  });
}

// ─── Mutation Hooks ───────────────────────────────────────────────────────────

/** PUT /v1/console/graph/ontology */
export function useUpdateOntology() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (schema: OntologySchema) => graphService.updateOntology(schema),
    onSuccess:  () => qc.invalidateQueries({ queryKey: keys.ontology() }),
  });
}

/** POST /v1/console/graph/query — Cypher query */
export function useGraphQuery() {
  return useMutation({
    mutationFn: (cypher: string) => graphService.query(cypher),
  });
}
