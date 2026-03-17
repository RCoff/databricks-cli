# NEXT CHANGELOG

## Release v0.295.0

### Notable Changes

- Add `bundle.engine` config setting to select the deployment engine (`terraform` or `direct`). The `DATABRICKS_BUNDLE_ENGINE` environment variable takes precedence over this setting. When the configured engine doesn't match existing deployment state, a warning is issued and the existing engine is used ([#4749](https://github.com/databricks/cli/pull/4749)).

### CLI

### Bundles
* Add `random_strings` configuration to generate stable random strings for bundles. Values are generated once using `crypto/rand` and persisted across deployments. Reference via `${random_string.<name>}`. Supports `length`, `special`, `upper`, `lower`, `numeric`, `min_*`, and `override_special` options, similar to Terraform's `random_string` resource.
* engine/direct: Fix permanent drift on experiment name field ([#4627](https://github.com/databricks/cli/pull/4627))
* engine/direct: Fix permissions state path to match input config schema ([#4703](https://github.com/databricks/cli/pull/4703))

### Dependency updates

### API Changes
