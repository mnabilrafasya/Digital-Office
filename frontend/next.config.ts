import type { NextConfig } from "next";
import path from "path";

const nextConfig: NextConfig = {
  // ── Catatan: proxy /api/v1/* ditangani oleh route handler ──
  // app/api/v1/[...path]/route.ts membaca cookie "token" dan meneruskan
  // request ke backend Go dengan header Authorization yang benar.

  reactCompiler: true,
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
    root: path.resolve(__dirname, ".."),
  },
};

export default nextConfig;

