# OLM bundle assets

Place the official Krkn icon at `config/olm/assets/krkn.svg`.

The icon should be square, preferably with a `256x256` viewBox. SVG is
preferred; PNG or JPG can also be injected by the bundle renderer when the
corresponding media type is configured. The release pipeline base64-encodes the
asset into `spec.icon` in the generated CSV, so the source asset does not need
to be committed into the generated bundle.
