import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createAdminOrganization,
  createAdminUser,
  listAdminOrganizations,
  listAdminRoles,
  listAdminUsers,
  resetAdminPassword,
  setAdminUserRoles,
  updateAdminOrganization,
  updateAdminUser,
  type AdminUserListQuery,
  type CreateAdminOrgInput,
  type CreateAdminUserInput,
  type UpdateAdminUserInput,
} from "@/services/admin";

export const adminKeys = {
  all: ["admin"] as const,
  users: () => [...adminKeys.all, "users"] as const,
  userList: (query: AdminUserListQuery) =>
    [...adminKeys.users(), query] as const,
  roles: () => [...adminKeys.all, "roles"] as const,
  orgs: () => [...adminKeys.all, "organizations"] as const,
};

export function useAdminUsers(query: AdminUserListQuery, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: adminKeys.userList(query),
    queryFn: () => listAdminUsers(query),
    enabled: options?.enabled ?? true,
  });
}

export function useAdminRoles() {
  return useQuery({
    queryKey: adminKeys.roles(),
    queryFn: () => listAdminRoles(),
  });
}

function useInvalidateAdminUsers() {
  const queryClient = useQueryClient();
  return () => {
    void queryClient.invalidateQueries({ queryKey: adminKeys.users() });
  };
}

export function useCreateAdminUser() {
  const invalidate = useInvalidateAdminUsers();
  return useMutation({
    mutationFn: (input: CreateAdminUserInput) => createAdminUser(input),
    onSuccess: () => invalidate(),
  });
}

export function useUpdateAdminUser() {
  const invalidate = useInvalidateAdminUsers();
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateAdminUserInput }) =>
      updateAdminUser(id, input),
    onSuccess: () => invalidate(),
  });
}

export function useSetAdminUserRoles() {
  const invalidate = useInvalidateAdminUsers();
  return useMutation({
    mutationFn: ({ id, role_ids }: { id: string; role_ids: string[] }) =>
      setAdminUserRoles(id, role_ids),
    onSuccess: () => invalidate(),
  });
}

export function useResetAdminPassword() {
  return useMutation({
    mutationFn: ({ id, new_password }: { id: string; new_password: string }) =>
      resetAdminPassword(id, new_password),
  });
}

export function useAdminOrganizations() {
  return useQuery({
    queryKey: adminKeys.orgs(),
    queryFn: () => listAdminOrganizations(),
  });
}

export function useCreateAdminOrganization() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateAdminOrgInput) => createAdminOrganization(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminKeys.orgs() });
    },
  });
}

export function useUpdateAdminOrganization() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      updateAdminOrganization(id, name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminKeys.orgs() });
    },
  });
}
