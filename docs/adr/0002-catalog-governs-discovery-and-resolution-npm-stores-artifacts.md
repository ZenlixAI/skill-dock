# Catalog Governs Discovery And Resolution; NPM Stores Artifacts

V1 separates artifact storage from product truth: the npm-compatible repository stores bundle artifacts, while the Catalog is the source of truth for discovery, version visibility, install resolution, and governance state. A release is considered published only after catalog registration succeeds, which lets the system distinguish published releases from orphaned artifacts and keeps lifecycle policy out of the package repository.
