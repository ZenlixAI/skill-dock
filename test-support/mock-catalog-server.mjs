import { createServer } from 'node:http';
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';

function basenameFromPackageName(packageName) {
  return packageName.split('/').at(-1) || packageName;
}

function sha512Integrity(buffer) {
  return `sha512-${createHash('sha512').update(buffer).digest('base64')}`;
}

function sha256Digest(buffer) {
  return `sha256:${createHash('sha256').update(buffer).digest('hex')}`;
}

function createTarballFixture({ packageName, version, description }) {
  const root = mkdtempSync(join(tmpdir(), 'skills-mock-catalog-'));
  const packageDir = join(root, 'package');
  mkdirSync(packageDir, { recursive: true });

  const skillMd = `---\nname: ${basenameFromPackageName(packageName)}\ndescription: ${description}\n---\n\n# ${basenameFromPackageName(packageName)}\n`;
  writeFileSync(join(packageDir, 'SKILL.md'), skillMd);
  writeFileSync(
    join(packageDir, 'package.json'),
    JSON.stringify(
      {
        name: packageName,
        version,
      },
      null,
      2
    )
  );

  const tarballPath = join(root, `${basenameFromPackageName(packageName)}-${version}.tgz`);
  execFileSync('tar', ['-czf', tarballPath, '-C', root, 'package']);
  const tarball = readFileSync(tarballPath);

  return {
    tarball,
    tarballPath,
    integrity: sha512Integrity(tarball),
    manifestDigest: sha256Digest(Buffer.from(skillMd)),
    cleanup() {
      rmSync(root, { recursive: true, force: true });
    },
  };
}

function createSkillDataset() {
  const agentNotesDirect = createTarballFixture({
    packageName: '@platform.skill/agent-notes',
    version: '1.0.0',
    description: 'Agent notes direct bundle',
  });
  const agentNotesProxy = createTarballFixture({
    packageName: '@platform.skill/agent-notes',
    version: '1.0.1',
    description: 'Agent notes proxy bundle',
  });
  const pdfSkill = createTarballFixture({
    packageName: '@platform.skill/pdf-skill',
    version: '1.2.0',
    description: 'PDF processing workflow',
  });

  const dataset = {
    packages: {
      '@platform.skill/agent-notes': {
        packageName: '@platform.skill/agent-notes',
        displayName: 'Agent Notes',
        description: 'Detail view',
        latestVersion: '1.0.1',
        status: 'active',
        sourceType: 'native',
        createdAt: '2026-01-01T00:00:00Z',
        updatedAt: '2026-01-02T00:00:00Z',
        versions: {
          '1.0.0': {
            packageName: '@platform.skill/agent-notes',
            version: '1.0.0',
            versionStatus: 'published',
            installPolicy: 'resolvable',
            publishedAt: '2026-01-01T00:00:00Z',
            registry: {
              npmRegistry: '',
              tarballUrl: '',
              integrity: '',
              shasum: 'shasum-direct',
              publishedAt: '2026-01-01T00:00:00Z',
              distTags: ['latest'],
            },
            bundle: {
              skillManifestDigest: agentNotesDirect.manifestDigest,
              components: ['commands'],
            },
            installTarget: {
              packageName: '@platform.skill/agent-notes',
              version: '1.0.0',
              npmRegistry: '',
              tarballUrl: '',
              integrity: agentNotesDirect.integrity,
              skillManifestDigest: agentNotesDirect.manifestDigest,
              versionStatus: 'published',
              registryAccessMode: 'direct',
            },
            tarballPath: '/direct/@platform.skill/agent-notes/-/agent-notes-1.0.0.tgz',
          },
          '1.0.1': {
            packageName: '@platform.skill/agent-notes',
            version: '1.0.1',
            versionStatus: 'published',
            installPolicy: 'resolvable',
            publishedAt: '2026-01-02T00:00:00Z',
            registry: {
              npmRegistry: '',
              tarballUrl: '',
              integrity: '',
              shasum: 'shasum-proxy',
              publishedAt: '2026-01-02T00:00:00Z',
              distTags: ['latest'],
            },
            bundle: {
              skillManifestDigest: agentNotesProxy.manifestDigest,
              components: ['commands', 'scripts'],
            },
            installTarget: {
              packageName: '@platform.skill/agent-notes',
              version: '1.0.1',
              npmRegistry: '',
              tarballUrl: '',
              integrity: agentNotesProxy.integrity,
              skillManifestDigest: agentNotesProxy.manifestDigest,
              versionStatus: 'published',
              registryAccessMode: 'proxy',
            },
            tarballPath: '/repository/skills-npm/@platform.skill/agent-notes/-/agent-notes-1.0.1.tgz',
          },
        },
      },
      '@platform.skill/pdf-skill': {
        packageName: '@platform.skill/pdf-skill',
        displayName: 'PDF Skill',
        description: 'PDF processing workflow',
        latestVersion: '1.2.0',
        status: 'active',
        sourceType: 'native',
        createdAt: '2026-01-01T00:00:00Z',
        updatedAt: '2026-01-02T00:00:00Z',
        versions: {
          '1.2.0': {
            packageName: '@platform.skill/pdf-skill',
            version: '1.2.0',
            versionStatus: 'published',
            installPolicy: 'resolvable',
            publishedAt: '2026-01-02T00:00:00Z',
            registry: {
              npmRegistry: '',
              tarballUrl: '',
              integrity: '',
              shasum: 'shasum-pdf',
              publishedAt: '2026-01-02T00:00:00Z',
              distTags: ['latest'],
            },
            bundle: {
              skillManifestDigest: pdfSkill.manifestDigest,
              components: ['commands', 'scripts'],
            },
            installTarget: {
              packageName: '@platform.skill/pdf-skill',
              version: '1.2.0',
              npmRegistry: '',
              tarballUrl: '',
              integrity: pdfSkill.integrity,
              skillManifestDigest: pdfSkill.manifestDigest,
              versionStatus: 'published',
              registryAccessMode: 'proxy',
            },
            tarballPath: '/repository/skills-npm/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz',
          },
        },
      },
    },
    tarballs: {
      '/direct/@platform.skill/agent-notes/-/agent-notes-1.0.0.tgz': {
        tarball: agentNotesDirect.tarball,
        version: '1.0.0',
        packageName: '@platform.skill/agent-notes',
      },
      '/repository/skills-npm/@platform.skill/agent-notes/-/agent-notes-1.0.1.tgz': {
        tarball: agentNotesProxy.tarball,
        version: '1.0.1',
        packageName: '@platform.skill/agent-notes',
      },
      '/repository/skills-npm/@platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz': {
        tarball: pdfSkill.tarball,
        version: '1.2.0',
        packageName: '@platform.skill/pdf-skill',
      },
    },
    requests: [],
    cleanup() {
      agentNotesDirect.cleanup();
      agentNotesProxy.cleanup();
      pdfSkill.cleanup();
    },
  };

  for (const pkg of Object.values(dataset.packages)) {
    for (const version of Object.values(pkg.versions)) {
      version.registry.npmRegistry = 'http://127.0.0.1:18080/repository/skills-npm/';
      version.registry.tarballUrl = `http://127.0.0.1:18080${version.tarballPath}`;
      version.installTarget.npmRegistry = 'http://127.0.0.1:18080/repository/skills-npm/';
      version.installTarget.tarballUrl = `http://127.0.0.1:18080${version.tarballPath}`;
    }
  }

  return dataset;
}

function jsonResponse(payload, status = 200) {
  return new Response(JSON.stringify(payload), {
    status,
    headers: { 'content-type': 'application/json' },
  });
}

function binaryResponse(buffer, headers = {}, status = 200) {
  return new Response(buffer, {
    status,
    headers: {
      'content-type': 'application/octet-stream',
      ...headers,
    },
  });
}

export function startMockCatalogServer({ port = 0 } = {}) {
  const dataset = createSkillDataset();
  const requests = dataset.requests;

  const server = createServer((request, response) => {
    const url = new URL(request.url || '/', `http://${request.headers.host || '127.0.0.1'}`);
    const { pathname, searchParams } = url;

    const origin = request.headers.origin || '';
    const corsHeaders = origin
      ? {
          'access-control-allow-origin': origin,
          'access-control-allow-credentials': 'true',
          vary: 'Origin',
        }
      : {};

    if (request.method === 'OPTIONS') {
      response.writeHead(204, {
        ...corsHeaders,
        'access-control-allow-methods': 'GET,POST,PUT,OPTIONS',
        'access-control-allow-headers':
          request.headers['access-control-request-headers'] ||
          'content-type, authorization, x-role, x-actor',
        'access-control-max-age': '600',
      });
      response.end();
      return;
    }

    if (request.method === 'GET' && pathname === '/api/v1/session') {
      response.writeHead(200, { 'content-type': 'application/json', ...corsHeaders });
      response.end(
        JSON.stringify({
          role: 'reader',
          actor: 'public-reader',
          authMechanism: 'public-readonly',
          allowedActions: ['session.read', 'catalog.read'],
          allowedPackages: [],
        })
      );
      return;
    }

    if (request.method === 'GET' && pathname === '/api/v1/skills') {
      const q = (searchParams.get('q') || '').trim().toLowerCase();
      const items = Object.values(dataset.packages)
        .filter((pkg) => !q || pkg.packageName.toLowerCase().includes(q) || pkg.displayName.toLowerCase().includes(q))
        .map((pkg) => ({
          packageName: pkg.packageName,
          displayName: pkg.displayName,
          description: pkg.description,
          latestVersion: pkg.latestVersion,
          sourceType: pkg.sourceType,
          status: pkg.status,
          latestVersionStatus: 'published',
          updatedAt: pkg.updatedAt,
        }));

      response.writeHead(200, { 'content-type': 'application/json', ...corsHeaders });
      response.end(
        JSON.stringify({
          items,
          pageInfo: {
            nextCursor: null,
            hasNextPage: false,
          },
        })
      );
      return;
    }

    if (request.method === 'GET' && pathname === '/api/v1/skills/curated') {
      response.writeHead(200, { 'content-type': 'application/json', ...corsHeaders });
      response.end(JSON.stringify({ items: [], pageInfo: { nextCursor: null, hasNextPage: false } }));
      return;
    }

    if (request.method === 'GET' && pathname === '/api/v1/governance-changes') {
      response.writeHead(200, { 'content-type': 'application/json', ...corsHeaders });
      response.end(JSON.stringify({ items: [], pageInfo: { nextCursor: null, hasNextPage: false } }));
      return;
    }

    const skillMatch = pathname.match(/^\/api\/v1\/skills\/([^/]+)(?:\/versions(?:\/([^/]+))?)?(?:\/install-target)?$/);
    if (request.method === 'GET' && skillMatch) {
      const packageName = decodeURIComponent(skillMatch[1]);
      const version = skillMatch[2] ? decodeURIComponent(skillMatch[2]) : null;
      const pkg = dataset.packages[packageName];
      if (!pkg) {
        response.writeHead(404, { 'content-type': 'application/json', ...corsHeaders });
        response.end(JSON.stringify({ error: { code: 'skill_not_found', message: 'Skill not found' } }));
        return;
      }

      if (pathname.endsWith('/versions') && !version) {
        const items = Object.values(pkg.versions)
          .slice()
          .sort((left, right) => String(right.publishedAt).localeCompare(String(left.publishedAt)))
          .map((detail) => ({
            packageName: detail.packageName,
            version: detail.version,
            versionStatus: detail.versionStatus,
            installPolicy: detail.installPolicy,
            publishedAt: detail.publishedAt,
          }));

        response.writeHead(200, { 'content-type': 'application/json', ...corsHeaders });
        response.end(
          JSON.stringify({
            items,
            pageInfo: {
              nextCursor: null,
              hasNextPage: false,
            },
          })
        );
        return;
      }

      if (pathname.endsWith('/install-target')) {
        const targetVersion = (searchParams.get('version') || version || pkg.latestVersion);
        const detail = pkg.versions[targetVersion];
        if (!detail) {
          response.writeHead(404, { 'content-type': 'application/json', ...corsHeaders });
          response.end(
            JSON.stringify({ error: { code: 'registry_version_not_found', message: 'Version not found' } })
          );
          return;
        }
        response.writeHead(200, { 'content-type': 'application/json', ...corsHeaders });
        response.end(
          JSON.stringify({
            packageName: detail.installTarget.packageName,
            version: detail.installTarget.version,
            npmRegistry: detail.installTarget.npmRegistry,
            tarballUrl: detail.installTarget.tarballUrl,
            integrity: detail.installTarget.integrity,
            skillManifestDigest: detail.installTarget.skillManifestDigest,
            versionStatus: detail.installTarget.versionStatus,
            registryAccessMode: detail.installTarget.registryAccessMode,
          })
        );
        return;
      }

      if (version) {
        const detail = pkg.versions[version];
        if (!detail) {
          response.writeHead(404, { 'content-type': 'application/json', ...corsHeaders });
          response.end(JSON.stringify({ error: { code: 'version_not_found', message: 'Version not found' } }));
          return;
        }
        response.writeHead(200, { 'content-type': 'application/json', ...corsHeaders });
        response.end(
          JSON.stringify({
            packageName: detail.packageName,
            version: detail.version,
            versionStatus: detail.versionStatus,
            installPolicy: detail.installPolicy,
            publishedAt: detail.publishedAt,
            registry: detail.registry,
            bundle: detail.bundle,
          })
        );
        return;
      }

      response.writeHead(200, { 'content-type': 'application/json', ...corsHeaders });
      response.end(
        JSON.stringify({
          packageName: pkg.packageName,
          displayName: pkg.displayName,
          description: pkg.description,
          sourceType: pkg.sourceType,
          status: pkg.status,
          latestVersion: pkg.latestVersion,
          latestVersionStatus: 'published',
          createdAt: pkg.createdAt,
          updatedAt: pkg.updatedAt,
        })
      );
      return;
    }

    if (request.method === 'GET' && dataset.tarballs[pathname]) {
      const tarball = dataset.tarballs[pathname];
      requests.push({
        path: pathname,
        authorization: request.headers.authorization || null,
        packageName: tarball.packageName,
        version: tarball.version,
      });
      response.writeHead(200, {
        'content-type': 'application/octet-stream',
        ...corsHeaders,
      });
      response.end(tarball.tarball);
      return;
    }

    response.writeHead(404, { 'content-type': 'application/json', ...corsHeaders });
    response.end(JSON.stringify({ error: { code: 'not_found', message: `No mock route for ${pathname}` } }));
  });

  return new Promise((resolve) => {
    server.listen(port, '127.0.0.1', () => {
      const address = server.address();
      const resolvedPort = typeof address === 'object' && address ? address.port : port;
      const baseUrl = `http://127.0.0.1:${resolvedPort}`;

      for (const pkg of Object.values(dataset.packages)) {
        for (const version of Object.values(pkg.versions)) {
          version.registry.npmRegistry = `${baseUrl}/repository/skills-npm/`;
          version.registry.tarballUrl = `${baseUrl}${version.tarballPath}`;
          version.installTarget.npmRegistry = `${baseUrl}/repository/skills-npm/`;
          version.installTarget.tarballUrl = `${baseUrl}${version.tarballPath}`;
        }
      }

      resolve({
        server,
        port: resolvedPort,
        baseUrl,
        requests,
        dataset,
        close: () =>
          new Promise((done) => {
            server.close(() => {
              dataset.cleanup();
              done();
            });
          }),
      });
    });
  });
}

if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  const port = Number(process.env.PORT || process.env.MOCK_CATALOG_PORT || 18080);
  const mode = process.env.MOCK_INSTALL_TARGET_MODE || 'proxy';
  void startMockCatalogServer({ port }).then(({ baseUrl }) => {
    // Keep process alive for the caller; log the URL for debugging.
    console.log(`mock catalog server listening on ${baseUrl} (${mode})`);
  });
}
