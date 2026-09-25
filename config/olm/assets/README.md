# OLM bundle assets

Place the official Krkn icon at `config/olm/assets/krkn.png`.

The icon should be square, preferably `256x256`, with a transparent background.
PNG is the default because it is rendered consistently by the OpenShift
Software Catalog. SVG or JPG can still be selected with `ICON_FILE`. The
release pipeline base64-encodes the asset into `spec.icon` in the generated
CSV, so the source asset does not need to be committed into the generated
bundle. The automated community catalog submission also copies the icon into
the FBC `olm.package` metadata used by the OpenShift Software Catalog.
