# Examples

This directory contains examples used for generated registry documentation and validation against the development provider.

The document generation tool looks for files in the following locations by default. All other *.tf files besides the ones mentioned below are ignored by the documentation tool. This is useful for creating examples that can run and/or ar testable even if some parts are not relevant for the documentation.

- **provider/provider.tf** is the provider index example.
- **data-sources/<full data source name>/data-source.tf** is a data source example.
- **resources/<full resource name>/resource.tf** is a resource example.
