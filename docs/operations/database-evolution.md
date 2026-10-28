# Database Evolution and Safe Upgrade Policy

## Present state

- `server/database/schema/current.sql` is the complete bootstrap schema; `foundation_schema_meta` stores the current schema marker.
- Local Compose, production Compose and integration CI currently use **MySQL 5.7**, with Redis separately. The exact production image must not be silently changed on an existing data volume.
- Data is logically owned by the modules listed in [Module boundaries](../architecture/module-boundaries.md), even though it lives in a single database.
- The bootstrap schema is suitable for disposable development environments and an intentionally initialized deployment. It is **not a verified in-place migration procedure** for populated production data.
- MySQL 5.7 is beyond extended support; an upgrade is a release task requiring testing and verified data preservation, not a find-and-replace in Compose.

## Minimum upgrade procedure before a populated production release

1. Inventory the deployed MySQL version, authentication plugin, SQL mode, charset/collation, constraints, row counts, data volume, backup/restore and rollback objective.
2. Create and **restore-test** a consistent backup in a disposable isolated environment. Verify table counts, key referential relationships and applications against the restored snapshot.
3. Establish an ordered, committed, versioned forward-migration history for production upgrades. Each migration records an owner, required preceding schema version, compatibility/locking assumptions and failure/rollback procedure. Do not continue treating complete `current.sql` as a safe upgrade script.
4. Test the target supported MySQL LTS version against this repository's schema, data, drivers, integration suite, read/write paths, constraint semantics, indexes and collation. Include a clean deployment and a representative populated-data migration.
5. Make application/database changes in a backwards-compatible order where multiple versions coexist. Keep destructive data changes separate from the first code rollout. Rehearse startup, shutdown and rollback.
6. Rehearse cutover and restoration with a copy of production data, acquire explicit maintenance/change approval, and monitor error rates, DB locks, slow queries and readiness after release.
7. Change the production image/configuration only in the specific verified upgrade goal. Do not reset or replace a volume to work around an incompatible data directory.

For now, deliberately retain the existing MySQL 5.7 test/deployment baseline. This document creates **no claim** that an upgrade or non-destructive migration has already happened.

## Schema ownership in a modular monolith

The schema file remains centralized as an assembly artifact; each module owns the definition and migrations for its own tables. New table prefixes must be registered in [Module boundaries](../architecture/module-boundaries.md) and the architecture regression test. Business modules must not directly query or change foreign module tables. A future service split can then move an owned dataset independently with an explicit data cutover.
