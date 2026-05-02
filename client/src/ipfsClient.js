'use strict';

const http = require('http');
const crypto = require('crypto');

const IPFS_HOST = process.env.IPFS_HOST || 'localhost';
const IPFS_PORT = parseInt(process.env.IPFS_PORT || '5001', 10);

// Uploads content to a local IPFS node and returns { cid, size }.
function addToIPFS(content) {
  const boundary = `----IPFSBoundary${Date.now()}`;
  const contentBuf = Buffer.isBuffer(content) ? content : Buffer.from(content, 'utf8');

  const body = Buffer.concat([
    Buffer.from(`--${boundary}\r\nContent-Disposition: form-data; name="file"; filename="document"\r\nContent-Type: application/octet-stream\r\n\r\n`),
    contentBuf,
    Buffer.from(`\r\n--${boundary}--\r\n`),
  ]);

  return new Promise((resolve, reject) => {
    const req = http.request(
      {
        hostname: IPFS_HOST,
        port: IPFS_PORT,
        path: '/api/v0/add?pin=true',
        method: 'POST',
        headers: {
          'Content-Type': `multipart/form-data; boundary=${boundary}`,
          'Content-Length': body.length,
        },
      },
      (res) => {
        let data = '';
        res.on('data', (chunk) => { data += chunk; });
        res.on('end', () => {
          try {
            const result = JSON.parse(data);
            resolve({ cid: result.Hash, size: result.Size });
          } catch (e) {
            reject(new Error(`IPFS add failed (is the IPFS node running?): ${data}`));
          }
        });
      }
    );
    req.on('error', (e) => reject(new Error(`IPFS connection error: ${e.message}. Start the IPFS node with: docker-compose -f network/docker/docker-compose-ipfs.yaml up -d`)));
    req.write(body);
    req.end();
  });
}

function computeSHA256(content) {
  const buf = Buffer.isBuffer(content) ? content : Buffer.from(content, 'utf8');
  return crypto.createHash('sha256').update(buf).digest('hex');
}

// Returns the HTTP gateway URL to retrieve a document by CID.
function getIPFSGatewayURL(cid) {
  return `http://${IPFS_HOST}:8080/ipfs/${cid}`;
}

module.exports = { addToIPFS, computeSHA256, getIPFSGatewayURL };
