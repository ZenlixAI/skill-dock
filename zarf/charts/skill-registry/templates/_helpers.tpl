{{- define "skill-registry.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "skill-registry.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s" (include "skill-registry.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "skill-registry.labels" -}}
app.kubernetes.io/name: {{ include "skill-registry.name" . }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "skill-registry.selectorLabels" -}}
app.kubernetes.io/name: {{ include "skill-registry.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "skill-registry.postgresName" -}}
{{- printf "%s-postgres" (include "skill-registry.fullname" .) -}}
{{- end -}}

{{- define "skill-registry.nexusName" -}}
{{- printf "%s-nexus" (include "skill-registry.fullname" .) -}}
{{- end -}}

{{- define "skill-registry.catalogName" -}}
{{- printf "%s-catalog" (include "skill-registry.fullname" .) -}}
{{- end -}}

{{- define "skill-registry.webName" -}}
{{- printf "%s-web" (include "skill-registry.fullname" .) -}}
{{- end -}}

{{- define "skill-registry.postgresDatabaseUrl" -}}
{{- if .Values.postgres.enabled -}}
{{- printf "postgres://%s:%s@%s.%s.svc.cluster.local:5432/%s?sslmode=disable" .Values.postgres.auth.user .Values.postgres.auth.password (include "skill-registry.postgresName" .) .Release.Namespace .Values.postgres.auth.database -}}
{{- else -}}
{{- printf "postgres://%s:%s@%s:%v/%s?sslmode=%s" .Values.postgres.auth.user .Values.postgres.auth.password .Values.database.host .Values.database.port .Values.postgres.auth.database .Values.database.sslmode -}}
{{- end -}}
{{- end -}}

{{- define "skill-registry.internalNpmRegistryUrl" -}}
{{- printf "http://%s.%s.svc.cluster.local:8081/repository/%s/" (include "skill-registry.nexusName" .) .Release.Namespace .Values.nexus.repositoryName -}}
{{- end -}}

{{- define "skill-registry.publicNpmRegistryUrl" -}}
{{- if .Values.nexus.publicRepositoryUrl -}}
{{- printf "%s/" (.Values.nexus.publicRepositoryUrl | trimSuffix "/") -}}
{{- else -}}
{{- printf "%s/repository/%s/" (include "skill-registry.nexusExternalBaseUrl" .) .Values.nexus.repositoryName -}}
{{- end -}}
{{- end -}}

{{- define "skill-registry.catalogUpstream" -}}
{{- printf "http://%s.%s.svc.cluster.local:%d" (include "skill-registry.catalogName" .) .Release.Namespace (int .Values.catalog.service.port) -}}
{{- end -}}

{{- define "skill-registry.webExternalBaseUrl" -}}
{{- printf "%s://%s" .Values.ingress.web.scheme .Values.ingress.web.host -}}
{{- end -}}

{{- define "skill-registry.nexusExternalBaseUrl" -}}
{{- printf "%s://%s" .Values.ingress.nexus.scheme .Values.ingress.nexus.host -}}
{{- end -}}

{{- define "skill-registry.traefikWebMatch" -}}
{{- if .Values.traefik.web.match -}}
{{- .Values.traefik.web.match -}}
{{- else -}}
{{- printf "Host(`%s`) && PathPrefix(`%s`)" .Values.ingress.web.host .Values.ingress.web.path -}}
{{- end -}}
{{- end -}}

{{- define "skill-registry.traefikNexusMatch" -}}
{{- if .Values.traefik.nexus.match -}}
{{- .Values.traefik.nexus.match -}}
{{- else -}}
{{- printf "Host(`%s`) && PathPrefix(`%s`)" .Values.ingress.nexus.host .Values.ingress.nexus.path -}}
{{- end -}}
{{- end -}}
