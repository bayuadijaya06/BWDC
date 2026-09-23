import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import {
  completeTask,
  createTask,
  fetchTask,
  listTasks,
  updateTask,
  type CreateTaskInput,
  type TaskListQuery,
  type UpdateTaskInput,
} from "@/services/tasks";

/**
 * Kunci kueri modul task, dikumpulkan di satu tempat supaya pembatalan
 * (`invalidateQueries`) tidak menebak bentuk kuncinya di komponen.
 *
 * Setiap perubahan status membatalkan **daftar** juga, bukan hanya detail:
 * `is_overdue` dan kolom Status di daftar berubah oleh transisi yang sama, dan
 * daftar yang tidak ikut disegarkan akan menampilkan tugas yang sudah selesai
 * sebagai masih terbuka.
 */
export const taskKeys = {
  all: ["tasks"] as const,
  lists: () => [...taskKeys.all, "list"] as const,
  list: (query: TaskListQuery) => [...taskKeys.lists(), query] as const,
  details: () => [...taskKeys.all, "detail"] as const,
  detail: (id: string) => [...taskKeys.details(), id] as const,
};

export function useTaskList(
  query: TaskListQuery,
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: taskKeys.list(query),
    queryFn: () => listTasks(query),
    placeholderData: keepPreviousData,
    enabled: options.enabled ?? true,
  });
}

export function useTask(id: string) {
  return useQuery({
    queryKey: taskKeys.detail(id),
    queryFn: () => fetchTask(id),
    enabled: id !== "",
  });
}

export function useCreateTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateTaskInput) => createTask(input),
    onSuccess: (task) => {
      queryClient.setQueryData(taskKeys.detail(task.id), task);
      void queryClient.invalidateQueries({ queryKey: taskKeys.lists() });
    },
  });
}

/**
 * Perubahan medan atau transisi `Start`/`Reopen`.
 *
 * Detailnya **diambil ulang**, bukan ditambal di cache: `updated_at` dan
 * `is_overdue` dihitung server, dan menebaknya di klien akan membuat dua angka
 * berbeda untuk tugas yang sama di dua halaman.
 */
export function useUpdateTask(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateTaskInput) => updateTask(id, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: taskKeys.detail(id) });
      void queryClient.invalidateQueries({ queryKey: taskKeys.lists() });
    },
  });
}

/** `Complete` — endpoint dan izin sendiri (`task:complete`), bukan `PATCH` status. */
export function useCompleteTask(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => completeTask(id),
    onSuccess: (task) => {
      queryClient.setQueryData(taskKeys.detail(id), task);
      void queryClient.invalidateQueries({ queryKey: taskKeys.lists() });
    },
  });
}
