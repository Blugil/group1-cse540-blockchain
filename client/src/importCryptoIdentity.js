'use strict';
/*
 * importCryptoIdentity.js
 * Imports the cryptogen-generated Admin identity into the wallet.
 * Use this when the network was started with cryptogen (not Fabric CA).
 */

const { Wallets } = require('fabric-network');
const fs = require('fs');
const path = require('path');

async function main() {
  const walletPath = path.join(__dirname, '..', 'wallet');
  const wallet = await Wallets.newFileSystemWallet(walletPath);
  console.log(`Wallet path: ${walletPath}`);

  const orgPath = path.resolve(
    __dirname, '..', '..', 'network', 'organizations',
    'peerOrganizations', 'manufacturer.shipment.com'
  );

  // Import Admin
  const adminKeyDir = path.join(orgPath, 'users', 'Admin@manufacturer.shipment.com', 'msp', 'keystore');
  const adminCertDir = path.join(orgPath, 'users', 'Admin@manufacturer.shipment.com', 'msp', 'signcerts');

  const adminKeyFile = fs.readdirSync(adminKeyDir)[0];
  const adminCertFile = fs.readdirSync(adminCertDir)[0];

  const adminKey = fs.readFileSync(path.join(adminKeyDir, adminKeyFile), 'utf8');
  const adminCert = fs.readFileSync(path.join(adminCertDir, adminCertFile), 'utf8');

  const adminIdentity = {
    credentials: { certificate: adminCert, privateKey: adminKey },
    mspId: 'ManufacturerMSP',
    type: 'X.509',
  };
  await wallet.put('admin', adminIdentity);
  console.log('Imported Admin@manufacturer.shipment.com as "admin"');

  // Import User1 as appUser
  const userKeyDir = path.join(orgPath, 'users', 'User1@manufacturer.shipment.com', 'msp', 'keystore');
  const userCertDir = path.join(orgPath, 'users', 'User1@manufacturer.shipment.com', 'msp', 'signcerts');

  const userKeyFile = fs.readdirSync(userKeyDir)[0];
  const userCertFile = fs.readdirSync(userCertDir)[0];

  const userKey = fs.readFileSync(path.join(userKeyDir, userKeyFile), 'utf8');
  const userCert = fs.readFileSync(path.join(userCertDir, userCertFile), 'utf8');

  const appUserIdentity = {
    credentials: { certificate: userCert, privateKey: userKey },
    mspId: 'ManufacturerMSP',
    type: 'X.509',
  };
  await wallet.put('appUser', appUserIdentity);
  console.log('Imported User1@manufacturer.shipment.com as "appUser"');
}

main().catch(e => { console.error(e); process.exit(1); });
