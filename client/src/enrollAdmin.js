'use strict';

const FabricCAServices = require('fabric-ca-client');
const { Wallets } = require('fabric-network');
const fs = require('fs');
const path = require('path');

async function main() {
  try {
    const ccpPath = path.resolve(
      __dirname,
      '..',
      '..',
      'network',
      'organizations',
      'peerOrganizations',
      'manufacturer.shipment.com',
      'connection-manufacturer.json'
    );
    const ccp = JSON.parse(fs.readFileSync(ccpPath, 'utf8'));

    const caInfo = ccp.certificateAuthorities['ca.manufacturer.shipment.com'];
    const caTLSCACerts = caInfo.tlsCACerts.pem;
    const ca = new FabricCAServices(
      caInfo.url,
      { trustedRoots: caTLSCACerts, verify: false },
      caInfo.caName
    );

    const walletPath = path.join(__dirname, '..', 'wallet');
    const wallet = await Wallets.newFileSystemWallet(walletPath);
    console.log(`Wallet path: ${walletPath}`);

    const identity = await wallet.get('admin');
    if (identity) {
      console.log('Admin user already exists in the wallet');
      return;
    }

    const enrollment = await ca.enroll({
      enrollmentID: 'admin',
      enrollmentSecret: 'adminpw',
    });

    const x509Identity = {
      credentials: {
        certificate: enrollment.certificate,
        privateKey: enrollment.key.toBytes(),
      },
      mspId: 'ManufacturerMSP',
      type: 'X.509',
    };

    await wallet.put('admin', x509Identity);
    console.log('Successfully enrolled admin user and stored in wallet');
  } catch (error) {
    console.error(`Failed to enroll admin user: ${error}`);
    process.exit(1);
  }
}

main();
