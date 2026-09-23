import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { UserProfile } from "@/services/auth";
import { ApiError } from "@/services/http";
import { useAuthStore } from "@/store/auth";
import { runAxe } from "@/test/a11y";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  list: vi.fn(),
  fetch: vi.fn(),
  archive: vi.fn(),
  update: vi.fn(),
  addMember: vi.fn(),
  removeMember: vi.fn(),
}));

vi.mock("@/services/projects", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/services/projects")>();
  return {
    ...actual,
    listProjects: mocks.list,
    createProject: mocks.create,
    fetchProject: mocks.fetch,
    archiveProject: mocks.archive,
    updateProject: mocks.update,
    addProjectMember: mocks.addMember,
    removeProjectMember: mocks.removeMember,
  };
});

const { CreateProjectDialog } = await import("./CreateProjectDialog");

const profile: UserProfile = {
  id: "u-owner",
  username: "admin",
  email: "admin@example.test",
  roles: ["administrator"],
  organization_id: "org-1",
  is_active: true,
  permissions: ["project:read", "project:create"],
};

const onCreated = vi.fn();
const onClose = vi.fn();

beforeEach(() => {
  mocks.create.mockReset();
  onCreated.mockReset();
  onClose.mockReset();
  useAuthStore.setState({
    status: "authenticated",
    profile,
    error: null,
    pending: false,
  });
});

function renderDialog() {
  return renderWithProviders(
    <CreateProjectDialog onClose={onClose} onCreated={onCreated} />,
  );
}

async function fillValidForm(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText("Kode"), "WEB");
  await user.type(screen.getByLabelText("Nama"), "Website Redesign");
}

describe("dialog buat project", () => {
  it("menolak form kosong di klien tanpa memanggil server", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: "Buat project" }));

    expect(screen.getByText("Kode project wajib diisi.")).toBeInTheDocument();
    expect(screen.getByText("Nama project wajib diisi.")).toBeInTheDocument();
    expect(mocks.create).not.toHaveBeenCalled();
  });

  it("menolak target selesai yang mendahului tanggal mulai", async () => {
    const user = userEvent.setup();
    renderDialog();

    await fillValidForm(user);
    // `input type=date` disetel utuh, bukan diketik karakter demi karakter:
    // jsdom menolak nilai antara yang belum berbentuk tanggal sah.
    fireEvent.change(screen.getByLabelText("Tanggal mulai"), {
      target: { value: "2026-10-10" },
    });
    fireEvent.change(screen.getByLabelText("Target selesai"), {
      target: { value: "2026-10-01" },
    });
    await user.click(screen.getByRole("button", { name: "Buat project" }));

    expect(
      screen.getByText("Target selesai tidak boleh mendahului tanggal mulai."),
    ).toBeInTheDocument();
    expect(mocks.create).not.toHaveBeenCalled();
  });

  it("mengirim kode apa adanya dan pemilik dari sesi yang sedang masuk", async () => {
    const user = userEvent.setup();
    mocks.create.mockResolvedValue({ project: { id: "p9" } });
    renderDialog();

    await user.type(screen.getByLabelText("Kode"), "web");
    await user.type(screen.getByLabelText("Nama"), "Website");
    await user.click(screen.getByRole("button", { name: "Buat project" }));

    await waitFor(() => {
      expect(mocks.create).toHaveBeenCalledWith({
        code: "web",
        name: "Website",
        description: "",
        owner_id: "u-owner",
        start_date: null,
        target_end_date: null,
      });
    });
    expect(onCreated).toHaveBeenCalledWith("p9");
  });

  it("meletakkan galat 422 pada field yang disebut server", async () => {
    const user = userEvent.setup();
    mocks.create.mockRejectedValue(
      new ApiError({
        status: 422,
        code: "VALIDATION_ERROR",
        message: "data tidak valid",
        details: [{ field: "code", error: "kode tidak cocok pola" }],
      }),
    );
    renderDialog();

    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Buat project" }));

    expect(await screen.findByText("kode tidak cocok pola")).toBeInTheDocument();
    expect(screen.getByLabelText("Kode")).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    expect(onCreated).not.toHaveBeenCalled();
  });

  it("menampilkan pesan 409 apa adanya, karena hanya satu sebabnya", async () => {
    const user = userEvent.setup();
    mocks.create.mockRejectedValue(
      new ApiError({
        status: 409,
        code: "CONFLICT",
        message: "kode project sudah dipakai di organisasi ini",
      }),
    );
    renderDialog();

    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Buat project" }));

    expect(
      await screen.findByRole("alert"),
    ).toHaveTextContent("kode project sudah dipakai di organisasi ini");
  });

  it("lolos pemeriksaan axe", async () => {
    renderDialog();
    expect(await runAxe(document.body)).toEqual([]);
  });
});
