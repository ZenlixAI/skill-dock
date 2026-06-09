type InstallCommandInput = {
  packageName: string;
  version: string;
  npmRegistry: string;
};

export function buildInstallCommand({ packageName, version, npmRegistry }: InstallCommandInput) {
  return `npm install ${packageName}@${version} --registry=${npmRegistry}`;
}
