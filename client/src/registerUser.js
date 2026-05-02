/*
 * Register Application User
 * ==========================
 * Registers and enrolls a new application user with the Fabric CA,
 * then stores credentials in the local wallet.
 */

'use strict';

const { Wallets } = require('fabric-network');
const FabricCAServices = require('fabric-ca-client');
const fs = require('fs');
const path = require('path');

async function main() {
  try {
    // Load connection profile
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

    // Create CA client
    const caURL = ccp.certificateAuthorities['ca.manufacturer.shipment.com'].url;
    const ca = new FabricCAServices(caURL);

    // Create wallet
    const walletPath = path.join(__dirname, '..', 'wallet');
    const wallet = await Wallets.newFileSystemWallet(walletPath);
    console.log(`Wallet path: ${walletPath}`);

    // Check if user already exists
    const userIdentity = await wallet.get('appUser');
    if (userIdentity) {
      console.log('Application user "appUser" already exists in the wallet');
      return;
    }

    // Check if admin exists
    const adminIdentity = await wallet.get('admin');
    if (!adminIdentity) {
      console.log('Admin identity not found. Run "npm run enroll-admin" first.');
      return;
    }

    // Build admin user object for registering new users
    const provider = wallet.getProviderRegistry().getProvider(adminIdentity.type);
    const adminUser = await provider.getUserContext(adminIdentity, 'admin');

    // Register the user
    const secret = await ca.register(
      {
        affiliation: 'manufacturer.department1',
        enrollmentID: 'appUser',
        role: 'client',
      },
      adminUser
    );

    // Enroll the user
    const enrollment = await ca.enroll({
      enrollmentID: 'appUser',
      enrollmentSecret: secret,
    });

    const x509Identity = {
      credentials: {
        certificate: enrollment.certificate,
        privateKey: enrollment.key.toBytes(),
      },
      mspId: 'ManufacturerMSP',
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
