import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import {
  addProjectMember,
  archiveProject,
  createProject,
  fetchProject,
  listProjects,
  removeProjectMember,
  updateProject,
  type CreateProjectInput,
  type ProjectListQuery,
  type ProjectMember,
  type ProjectMemberRole,
  type UpdateProjectInput,
} from "@/services/projects";

/**
 * Kunci kueri modul project, dikumpulkan di satu tempat supaya pembatalan
 * (`invalidateQueries`) tidak menebak bentuk kuncinya di komponen.
 */
export const projectKeys = {
  all: ["projects"] as const,
  lists: () => [...projectKeys.all, "list"] as const,
  list: (query: ProjectListQuery) => [...projectKeys.lists(), query] as const,
  details: () => [...projectKeys.all, "detail"] as const,
  detail: (id: string) => [...projectKeys.details(), id] as const,
};

/**
 * Daftar project. `placeholderData: keepPreviousData` menjaga baris halaman
 * sebelumnya tetap tampil saat halaman berikutnya dimuat, sehingga tabel tidak
 * berkedip ke kerangka pemuatan setiap kali penyaring berubah.
 */
export function useProjectList(
  query: ProjectListQuery,
  /**
   * `enabled: false` dipakai pemanggil yang hanya **menumpang** daftar project
   * untuk mengisi pilihan penyaring, mis. penyaring project di halaman
   * Documents. Tanpa ini, peran yang tidak memegang `project:read` akan memicu
   * permintaan yang pasti dibalas `403` setiap kali daftar dokumen dibuka.
   */
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: projectKeys.list(query),
    queryFn: () => listProjects(query),
    placeholderData: keepPreviousData,
    enabled: options.enabled ?? true,
  });
}

/** Detail project beserta anggotanya (`GET /projects/:id`). */
export function useProject(id: string) {
  return useQuery({
    queryKey: projectKeys.detail(id),
    queryFn: () => fetchProject(id),
    enabled: id !== "",
  });
}

/**
 * Anggota satu project, dibaca dari detailnya.
 *
 * Dipakai sebagai **sumber pilihan penanggung jawab task**: kontrak belum
 * memuat endpoint daftar pengguna (kelas temuan C-063), jadi halaman task tidak
 * boleh mengarang pencarian pengguna. Ia hanya menyediakan orang yang benar-
 * benar terhubung ke project itu.
 *
 * Pemanggil dengan `id` kosong tidak memicu permintaan apa pun: `useProject`
 * sudah mematikan dirinya untuk id kosong, dan itu penting karena halaman task
 * tidak selalu punya project terpilih.
 */
export function useProjectMembers(id: string): {
  data: ProjectMember[] | undefined;
  isPending: boolean;
  isError: boolean;
} {
  const query = useProject(id);
  return {
    data: query.data?.members,
    isPending: query.isPending,
    isError: query.isError,
  };
}

export function useCreateProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateProjectInput) => createProject(input),
    onSuccess: (detail) => {
      // Detail project baru langsung tersedia, jadi halaman tujuan tidak
      // memanggil ulang `GET /projects/:id` yang baru saja dijawab server.
      queryClient.setQueryData(projectKeys.detail(detail.project.id), detail);
      void queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

export function useUpdateProject(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateProjectInput) => updateProject(id, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: projectKeys.detail(id) });
      void queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

export function useArchiveProject() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => archiveProject(id),
    onSuccess: (project) => {
      queryClient.setQueryData(
        projectKeys.detail(project.id),
        (previous: { members: unknown } | undefined) =>
          previous === undefined ? previous : { ...previous, project },
      );
      void queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

export function useAddProjectMember(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { user_id: string; role: ProjectMemberRole }) =>
      addProjectMember(id, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: projectKeys.detail(id) });
      void queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}

export function useRemoveProjectMember(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => removeProjectMember(id, userId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: projectKeys.detail(id) });
      void queryClient.invalidateQueries({ queryKey: projectKeys.lists() });
    },
  });
}
