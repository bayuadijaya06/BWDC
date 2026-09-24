import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  actWorkflowInstance,
  fetchWorkflowInstance,
  listWorkflowDefinitions,
  listWorkflowInstances,
  resubmitWorkflowInstance,
  submitWorkflowInstance,
  type SubmitWorkflowInput,
  type WorkflowActionInput,
  type WorkflowInstancesQuery,
} from "@/services/workflows";

export const workflowKeys = {
  all: ["workflows"] as const,
  instances: (query: WorkflowInstancesQuery) =>
    [...workflowKeys.all, "instances", query] as const,
  instance: (id: string) => [...workflowKeys.all, "instance", id] as const,
  definitions: () => [...workflowKeys.all, "definitions"] as const,
};

export function useWorkflowInstances(
  query: WorkflowInstancesQuery,
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: workflowKeys.instances(query),
    queryFn: () => listWorkflowInstances(query),
    enabled: options.enabled ?? true,
  });
}

export function useWorkflowInstance(id: string) {
  return useQuery({
    queryKey: workflowKeys.instance(id),
    queryFn: () => fetchWorkflowInstance(id),
    enabled: id !== "",
  });
}

export function useActWorkflowInstance() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: WorkflowActionInput }) =>
      actWorkflowInstance(id, input),
    onSuccess: (_data, variables) => {
      void client.invalidateQueries({ queryKey: workflowKeys.all });
      void client.invalidateQueries({
        queryKey: workflowKeys.instance(variables.id),
      });
    },
  });
}

export function useResubmitWorkflowInstance() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, version }: { id: string; version?: number }) =>
      resubmitWorkflowInstance(id, version),
    onSuccess: (_data, variables) => {
      void client.invalidateQueries({ queryKey: workflowKeys.all });
      void client.invalidateQueries({
        queryKey: workflowKeys.instance(variables.id),
      });
    },
  });
}

export function useWorkflowDefinitions(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: workflowKeys.definitions(),
    queryFn: () => listWorkflowDefinitions(),
    enabled: options.enabled ?? true,
  });
}

export function useSubmitWorkflowInstance() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: SubmitWorkflowInput) => submitWorkflowInstance(input),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: workflowKeys.all });
    },
  });
}
