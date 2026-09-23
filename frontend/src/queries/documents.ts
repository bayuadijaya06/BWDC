import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import {
  archiveDocument,
  createDocument,
  fetchDocument,
  listDocumentVersions,
  listDocuments,
  uploadDocumentVersion,
  type CreateDocumentInput,
  type DocumentListQuery,
  type UploadVersionInput,
} from "@/services/documents";

/**
 * Kunci kueri modul document, dikumpulkan di satu tempat supaya pembatalan
 * (`invalidateQueries`) tidak menebak bentuk kuncinya di komponen.
 *
 * Yang dibatalkan karena perubahan **versi** adalah `lists` juga, bukan hanya
 * `detail`: kolom Version dan Last Updated di daftar berubah oleh unggahan, dan
 * daftar yang tidak ikut disegarkan akan menampilkan versi lama.
 */
export const documentKeys = {
  all: ["documents"] as const,
  lists: () => [...documentKeys.all, "list"] as const,
  list: (query: DocumentListQuery) => [...documentKeys.lists(), query] as const,
  details: () => [...documentKeys.all, "detail"] as const,
  detail: (id: string) => [...documentKeys.details(), id] as const,
  versions: (id: string) => [...documentKeys.detail(id), "versions"] as const,
};

export function useDocumentList(
  query: DocumentListQuery,
  /**
   * `enabled: false` dipakai pemanggil yang **menumpang** daftar dokumen untuk
   * mengisi pilihan, mis. pemilih Related Document di form task. Ia tidak boleh
   * berjalan sebelum project dipilih: kueri tanpa `project_id` akan meminta
   * halaman dokumen yang tidak berhubungan dengan pilihan pengguna.
   */
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: documentKeys.list(query),
    queryFn: () => listDocuments(query),
    placeholderData: keepPreviousData,
    enabled: options.enabled ?? true,
  });
}

export function useDocument(id: string) {
  return useQuery({
    queryKey: documentKeys.detail(id),
    queryFn: () => fetchDocument(id),
    enabled: id !== "",
  });
}

export function useDocumentVersions(id: string) {
  return useQuery({
    queryKey: documentKeys.versions(id),
    queryFn: () => listDocumentVersions(id),
    enabled: id !== "",
  });
}

export function useCreateDocument() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateDocumentInput) => createDocument(input),
    onSuccess: (detail) => {
      queryClient.setQueryData(documentKeys.detail(detail.document.id), detail);
      void queryClient.invalidateQueries({ queryKey: documentKeys.lists() });
    },
  });
}

export function useUploadDocumentVersion(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: UploadVersionInput) => uploadDocumentVersion(id, input),
    onSuccess: () => {
      // Detail diambil ulang, bukan ditambal di cache: `latest_version`,
      // `current_version`, dan `updated_at` dihitung server, dan menebaknya di
      // klien berarti dua sumber kebenaran untuk penomoran versi.
      void queryClient.invalidateQueries({ queryKey: documentKeys.detail(id) });
      void queryClient.invalidateQueries({ queryKey: documentKeys.versions(id) });
      void queryClient.invalidateQueries({ queryKey: documentKeys.lists() });
    },
  });
}

export function useArchiveDocument() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { id: string; reason?: string }) =>
      archiveDocument(input.id, input.reason),
    onSuccess: (document) => {
      queryClient.setQueryData(
        documentKeys.detail(document.id),
        (previous: { current_version: unknown } | undefined) =>
          previous === undefined ? previous : { ...previous, document },
      );
      void queryClient.invalidateQueries({ queryKey: documentKeys.lists() });
    },
  });
}
