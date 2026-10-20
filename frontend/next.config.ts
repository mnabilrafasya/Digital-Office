import type { NextConfig } from "next";
import path from "path";

const nextConfig: NextConfig = {
  /* config options here */
  async redirects() {
    return [
      {
        source: "/",
        destination: "/login",
        permanent: false,
      },
    ];
  },
  // cacheComponents dan partialPrefetching dimatikan:
  // semua halaman bersifat per-user, caching tidak berguna dan
  // bisa menampilkan data stale setelah integrasi auth.
  // cacheComponents: false,   // default false
  // partialPrefetching: false, // default false
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
