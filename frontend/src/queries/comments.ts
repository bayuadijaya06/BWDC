import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createComment,
  deleteComment,
  listComments,
  updateComment,
  type CommentListQuery,
  type CreateCommentInput,
  type UpdateCommentInput,
} from "@/services/comments";

/**
 * Kunci kueri modul comment. Daftar selalu per entitas
 * (`42-API.md` §7: `?entity_type=&entity_id=` wajib), jadi kunci memakai
 * keduanya — pembatalan satu entitas tidak menyentuh entitas lain.
 */
export const commentKeys = {
  all: ["comments"] as const,
  lists: () => [...commentKeys.all, "list"] as const,
  list: (query: CommentListQuery) =>
    [...commentKeys.lists(), query] as const,
};

export function useCommentList(query: CommentListQuery) {
  return useQuery({
    queryKey: commentKeys.list(query),
    queryFn: () => listComments(query),
  });
}

export function useCreateComment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateCommentInput) => createComment(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: commentKeys.lists() });
    },
  });
}

export function useUpdateComment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { id: string } & UpdateCommentInput) =>
      updateComment(input.id, { content: input.content }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: commentKeys.lists() });
    },
  });
}

export function useDeleteComment() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteComment(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: commentKeys.lists() });
    },
  });
}
