/*
 * Registers and enrolls the application user with the Fabric CA, then saves to wallet.
 * Only needed when using a Fabric CA (not cryptogen). With the test-network, run
 * importCryptoIdentity.js instead.
 */

'use strict';

const { Wallets } = require('fabric-network');
const FabricCAServices = require('fabric-ca-client');
const fs = require('fs');
const path = require('path');
const { ccpPath } = require('./fabricConfig');

async function main() {
  try {
    const ccp = JSON.parse(fs.readFileSync(ccpPath, 'utf8'));

    const caURL = ccp.certificateAuthorities['ca.org1.example.com'].url;
    const ca = new FabricCAServices(caURL);

    const walletPath = path.join(__dirname, '..', 'wallet');
    const wallet = await Wallets.newFileSystemWallet(walletPath);
    console.log(`Wallet path: ${walletPath}`);

    const userIdentity = await wallet.get('appUser');
    if (userIdentity) {
      console.log('Application user "appUser" already exists in the wallet');
      return;
    }

    const adminIdentity = await wallet.get('admin');
    if (!adminIdentity) {
      console.log('Admin identity not found. Run "npm run enroll-admin" first.');
      return;
    }

    // admin user object is needed to authorize the registration request
    const provider = wallet.getProviderRegistry().getProvider(adminIdentity.type);
    const adminUser = await provider.getUserContext(adminIdentity, 'admin');

    const secret = await ca.register(
      {
        affiliation: 'org1.department1',
        enrollmentID: 'appUser',
        role: 'client',
      },
      adminUser
    );

    const enrollment = await ca.enroll({
      enrollmentID: 'appUser',
      enrollmentSecret: secret,
    });

    const x509Identity = {
      credentials: {
        certificate: enrollment.certificate,
        privateKey: enrollment.key.toBytes(),
      },
      mspId: 'Org1MSP',
      type: 'X.509',
    };

    await wallet.put('appUser', x509Identity);
    console.log('Successfully registered and enrolled user "appUser" and stored in wallet');
  } catch (error) {
    console.error(`Failed to register user: ${error}`);
    process.exit(1);
  }
}

main();
