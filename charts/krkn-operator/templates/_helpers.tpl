{{/*
Expand the name of the chart.
*/}}
{{- define "krkn-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "krkn-operator.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "krkn-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "krkn-operator.labels" -}}
helm.sh/chart: {{ include "krkn-operator.chart" . }}
{{ include "krkn-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "krkn-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "krkn-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "krkn-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "krkn-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Operator component fullname
*/}}
{{- define "krkn-operator.operator.fullname" -}}
{{ include "krkn-operator.fullname" . }}-operator
{{- end }}

{{/*
ACM component fullname
*/}}
{{- define "krkn-operator.acm.fullname" -}}
{{ include "krkn-operator.fullname" . }}-acm
{{- end }}

{{/*
Console component fullname
*/}}
{{- define "krkn-operator.console.fullname" -}}
{{ include "krkn-operator.fullname" . }}-console
{{- end }}

{{/*
Namespace to use
*/}}
{{- define "krkn-operator.namespace" -}}
{{- if .Values.global.namespaceOverride }}
{{- .Values.global.namespaceOverride }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}
