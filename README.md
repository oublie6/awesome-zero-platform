# Awesome Zero Platform

A modular Go application platform built on go-zero, designed for reusable server foundations and multiple client applications.

## Project layout

- `server/` — reusable server-side foundation and capability modules
- `clients/` — Vue 3, WeChat Mini Program, H5, and app clients
- `deploy/` — local, container, and Kubernetes deployment assets
- `docs/` — architecture and development documentation
- `scripts/` — project automation scripts

The project starts as a modular monolith and keeps module boundaries clear so high-load capabilities can be extracted into independent services later.
