{{/*
Common names and labels.
*/}}
{{- define "skill-dock.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "skill-dock.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "skill-dock.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{ include "skill-dock.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "skill-dock.selectorLabels" -}}
app.kubernetes.io/name: {{ include "skill-dock.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "skill-dock.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "skill-dock.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "skill-dock.postgresName" -}}
{{- printf "%s-postgres" (include "skill-dock.fullname" .) -}}
{{- end -}}

{{- define "skill-dock.verdaccioName" -}}
{{- printf "%s-verdaccio" (include "skill-dock.fullname" .) -}}
{{- end -}}

{{- define "skill-dock.catalogName" -}}
{{- printf "%s-catalog" (include "skill-dock.fullname" .) -}}
{{- end -}}

{{- define "skill-dock.siteName" -}}
{{- printf "%s-site" (include "skill-dock.fullname" .) -}}
{{- end -}}

{{- define "skill-dock.catalogSecretName" -}}
{{- printf "%s-catalog-secrets" (include "skill-dock.fullname" .) -}}
{{- end -}}

{{- define "skill-dock.siteSecretName" -}}
{{- printf "%s-site-secrets" (include "skill-dock.fullname" .) -}}
{{- end -}}

{{- define "skill-dock.verdaccioSecretName" -}}
{{- printf "%s-verdaccio-secrets" (include "skill-dock.fullname" .) -}}
{{- end -}}

{{- define "skill-dock.postgresSecretName" -}}
{{- printf "%s-postgres-secrets" (include "skill-dock.fullname" .) -}}
{{- end -}}

{{/*
Embedded component switches. External configuration always wins.
*/}}
{{- define "skill-dock.useEmbeddedPostgres" -}}
{{- if and .Values.postgres.enabled (not .Values.database.url) (not .Values.database.existingSecret) -}}true{{- end -}}
{{- end -}}

{{- define "skill-dock.useEmbeddedVerdaccio" -}}
{{- if and .Values.verdaccio.enabled (not .Values.artifactRepository.url) -}}true{{- end -}}
{{- end -}}

{{/*
Artifact repository URLs.
*/}}
{{- define "skill-dock.artifactRepositoryUrl" -}}
{{- if .Values.artifactRepository.url -}}
{{- printf "%s/" (.Values.artifactRepository.url | trimSuffix "/") -}}
{{- else if eq (include "skill-dock.useEmbeddedVerdaccio" .) "true" -}}
{{- printf "http://%s.%s.svc.cluster.local:%d/" (include "skill-dock.verdaccioName" .) .Release.Namespace (int .Values.verdaccio.service.port) -}}
{{- end -}}
{{- end -}}

{{- define "skill-dock.ingressBaseUrl" -}}
{{- $scheme := ternary "https" "http" .tls.enabled -}}
{{- printf "%s://%s%s" $scheme .host (trimSuffix "/" .path) -}}
{{- end -}}

{{- define "skill-dock.publicArtifactRepositoryUrl" -}}
{{- if .Values.artifactRepository.publicUrl -}}
{{- printf "%s/" (.Values.artifactRepository.publicUrl | trimSuffix "/") -}}
{{- else if .Values.artifactRepository.url -}}
{{- printf "%s/" (.Values.artifactRepository.url | trimSuffix "/") -}}
{{- else if and .Values.ingress.enabled .Values.site.enabled -}}
{{- printf "%s/repository/skills-npm/" (include "skill-dock.ingressBaseUrl" .Values.ingress.site | trimSuffix "/") -}}
{{- else if and .Values.ingress.enabled .Values.ingress.catalog.enabled -}}
{{- printf "%s/repository/skills-npm/" (include "skill-dock.ingressBaseUrl" .Values.ingress.catalog | trimSuffix "/") -}}
{{- else if and .Values.ingress.enabled .Values.ingress.verdaccio.enabled (eq (include "skill-dock.useEmbeddedVerdaccio" .) "true") -}}
{{- printf "%s/" (include "skill-dock.ingressBaseUrl" .Values.ingress.verdaccio | trimSuffix "/") -}}
{{- else -}}
{{- include "skill-dock.artifactRepositoryUrl" . -}}
{{- end -}}
{{- end -}}

{{- define "skill-dock.catalogUpstream" -}}
{{- printf "http://%s.%s.svc.cluster.local:%d" (include "skill-dock.catalogName" .) .Release.Namespace (int .Values.catalog.service.port) -}}
{{- end -}}

{{- define "skill-dock.traefikMatch" -}}
{{- printf "Host(`%s`) && PathPrefix(`%s`)" .host .path -}}
{{- end -}}

{{/*
Secret helpers. Generated values are reused from the existing release Secrets
on upgrade.
*/}}
{{- define "skill-dock.secretFrom" -}}
{{- $secret := index . 0 -}}
{{- $key := index . 1 -}}
{{- if and $secret $secret.data (hasKey $secret.data $key) -}}
{{- index $secret.data $key | b64dec -}}
{{- else -}}
{{- randAlphaNum 40 -}}
{{- end -}}
{{- end -}}

{{- define "skill-dock.secretOptional" -}}
{{- $secret := index . 0 -}}
{{- $key := index . 1 -}}
{{- if and $secret $secret.data (hasKey $secret.data $key) -}}
{{- index $secret.data $key | b64dec -}}
{{- end -}}
{{- end -}}

{{/*
Default Verdaccio configuration.
*/}}
{{- define "skill-dock.verdaccioConfig" -}}
{{- if and .Values.verdaccio.uplinks .Values.verdaccio.proxyUplink (not (hasKey .Values.verdaccio.uplinks .Values.verdaccio.proxyUplink)) -}}
{{- fail "verdaccio.proxyUplink must name an entry in verdaccio.uplinks" -}}
{{- end -}}
storage: /verdaccio/storage
listen: 0.0.0.0:4873
url_prefix: /
auth:
  htpasswd:
    file: /verdaccio/conf/htpasswd
    max_users: -1
publish:
  allow_offline: true
{{- if .Values.verdaccio.uplinks }}
uplinks:
{{- range $name, $url := .Values.verdaccio.uplinks }}
  {{ $name }}:
    url: {{ $url | quote }}
{{- end }}
{{- end }}
packages:
  '@*/*':
    access: {{ ternary "$all" "$authenticated" .Values.verdaccio.enableAnonymousRead }}
    publish: $authenticated
{{- if and .Values.verdaccio.uplinks .Values.verdaccio.proxyUplink }}
    proxy: {{ .Values.verdaccio.proxyUplink }}
{{- end }}
  '**':
    access: {{ ternary "$all" "$authenticated" .Values.verdaccio.enableAnonymousRead }}
    publish: $authenticated
{{- if and .Values.verdaccio.uplinks .Values.verdaccio.proxyUplink }}
    proxy: {{ .Values.verdaccio.proxyUplink }}
{{- end }}
web:
  enable: {{ .Values.verdaccio.web.enabled }}
log:
  - {type: stdout, format: pretty, level: http}
{{- end -}}
