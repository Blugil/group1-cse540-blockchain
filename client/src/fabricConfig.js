'use strict';

const path = require('path');
const fs   = require('fs');
const os   = require('os');

// Search common installation locations in order, then fall back to FABRIC_SAMPLES_PATH env var.
function findFabricSamples() {
  const candidates = [
    process.env.FABRIC_SAMPLES_PATH,
    path.join(os.homedir(), 'fabric-install', 'fabric-samples'),
    path.join(os.homedir(), 'fabric-samples'),
    path.join(os.homedir(), 'go', 'src', 'github.com', 'hyperledger', 'fabric-samples'),
    '/usr/local/fabric-samples',
    '/opt/fabric-samples',
  ].filter(Boolean);

  for (const p of candidates) {
    if (fs.existsSync(path.join(p, 'test-network', 'network.sh'))) {
      return p;
    }
  }

  throw new Error(
    '\nfabric-samples not found in any of the standard locations.\n' +
    'Set the FABRIC_SAMPLES_PATH environment variable to your installation:\n' +
    '  export FABRIC_SAMPLES_PATH=/path/to/fabric-samples\n' +
    'Then re-run the command.\n'
  );
}

const fabricSamplesPath = findFabricSamples();
const testNetworkPath   = path.join(fabricSamplesPath, 'test-network');
const org1Path          = path.join(testNetworkPath, 'organizations', 'peerOrganizations', 'org1.example.com');

module.exports = {
  fabricSamplesPath,
  testNetworkPath,
  org1Path,
  ccpPath: path.join(org1Path, 'connection-org1.json'),
};
