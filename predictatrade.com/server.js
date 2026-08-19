const fs = require('node:fs');
const path = require('node:path');
const http = require('node:http');
const tls = require('node:tls');
const net = require('node:net');
const zlib = require('node:zlib');
const root = __dirname;
const port = Number(process.env.PORT || 8080);
const smtpHost = process.env.SMTP_HOST || 'predictatrade.com';
const smtpPort = Number(process.env.SMTP_PORT || 465);
const smtpUser = process.env.SMTP_USER || 'no-reply@predictatrade.com';
const smtpPass = process.env.SMTP_PASS;
const smtpSecure = process.env.SMTP_SECURE !== 'false' && smtpPort === 465;

function reply(socket, expected) {
  return new Promise((resolve, reject) => {
    let buffer = '';
    const onData = (chunk) => {
      buffer += chunk.toString();
      const last = buffer.split(/\r?\n/).filter(Boolean).at(-1) || '';
      if (/^\d{3} /.test(last)) {
        socket.off('data', onData);
        const code = Number(last.slice(0, 3));
        if (!expected.includes(code)) reject(new Error(`SMTP ${code}`)); else resolve();
      }
    };
    socket.on('data', onData);
    socket.once('error', reject);
  });
}
async function command(socket, value, expected = [250]) { socket.write(`${value}\r\n`); await reply(socket, expected); }
async function sendMail({ name, email, message }) {
  if (!smtpPass) throw new Error('SMTP_PASS is not configured');
  let socket = smtpSecure ? tls.connect({ host: smtpHost, port: smtpPort, servername: smtpHost }) : net.connect({ host: smtpHost, port: smtpPort });
  await reply(socket, [220]);
  await command(socket, `EHLO ${smtpHost}`);
  if (!smtpSecure) {
    await command(socket, 'STARTTLS', [220]);
    socket = await new Promise((resolve, reject) => { const secure = tls.connect({ socket, host: smtpHost, servername: smtpHost }, () => resolve(secure)); secure.once('error', reject); });
    await command(socket, `EHLO ${smtpHost}`);
  }
  await command(socket, 'AUTH LOGIN', [334]);
  await command(socket, Buffer.from(smtpUser).toString('base64'), [334]);
  await command(socket, Buffer.from(smtpPass).toString('base64'), [235]);
  await command(socket, `MAIL FROM:<${smtpUser}>`);
  await command(socket, `RCPT TO:<${smtpUser}>`, [250, 251]);
  socket.write('DATA\r\n'); await reply(socket, [354]);
  const body = `Name: ${name}\r\nEmail: ${email}\r\n\r\n${message}`;
  socket.write(`From: ${smtpUser}\r\nTo: ${smtpUser}\r\nReply-To: ${email}\r\nSubject: Website contact from ${name}\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n${body.replace(/\r?\n/g, '\r\n')}\r\n.\r\n`);
  await reply(socket, [250]); socket.end('QUIT\r\n');
}
function send(res, status, data, type = 'application/json', request, cache = 'no-store') {
  const body = type === 'application/json' ? Buffer.from(JSON.stringify(data)) : Buffer.from(data);
  const canCompress = request && /\bgzip\b/.test(request.headers['accept-encoding'] || '') && /^(text\/|application\/(javascript|json|xml))/.test(type);
  const output = canCompress ? zlib.gzipSync(body, { level: 6 }) : body;
  res.writeHead(status, { 'Content-Type': type, 'Cache-Control': cache, ...(canCompress ? { 'Content-Encoding': 'gzip', Vary: 'Accept-Encoding' } : {}) });
  res.end(output);
}
const server = http.createServer((req, res) => {
  if (req.method === 'POST' && req.url === '/api/contact') {
    let raw = ''; req.on('data', (chunk) => { raw += chunk; if (raw.length > 20000) req.destroy(); });
    req.on('end', async () => { try { const data = JSON.parse(raw); const name = String(data.name || '').trim(); const email = String(data.email || '').trim(); const message = String(data.message || '').trim(); if (!name || !/^\S+@\S+\.\S+$/.test(email) || !message) return send(res, 400, { error: 'Invalid form data' }); await sendMail({ name, email, message }); send(res, 200, { ok: true }); } catch (error) { console.error(error.message); send(res, 500, { error: 'Unable to send message' }); } }); return;
  }
  const requested = req.url === '/' ? '/index.html' : req.url.split('?')[0];
  const file = path.join(root, requested.replace(/^\//, ''));
  if (!file.startsWith(root) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) return send(res, 404, 'Not found', 'text/plain');
  const types = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.xml': 'application/xml', '.txt': 'text/plain', '.webp': 'image/webp', '.png': 'image/png' };
  send(res, 200, fs.readFileSync(file), types[path.extname(file)] || 'application/octet-stream', req, path.basename(file) === 'index.html' ? 'no-cache, must-revalidate' : 'public, max-age=31536000, immutable');
});
server.listen(port, () => console.log(`Predict-A-Trade site listening on :${port}`));
