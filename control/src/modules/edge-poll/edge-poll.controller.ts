import { Body, Controller, Post, Headers, Req, UnauthorizedException } from '@nestjs/common';
import { EdgePollService } from './edge-poll.service';
import { DeviceAuthService } from '../device-auth/device-auth.service';

/**
 * EdgePollController — EA-direct transport endpoints.
 *
 * Auth: Proof-of-Device HMAC (same scheme as the Windows agent).
 *   Headers:
 *     X-Device-Id:        <device uuid>
 *     X-Device-Timestamp: <unix ms>
 *     X-Device-Nonce:     <random unique string>
 *     X-Device-Signature: hex(HMAC_SHA256(device_secret,
 *         "v1\n<timestamp>\n<nonce>\nPOST\n<path>\n<sha256(body)>\n<device_id>"))
 *   Path must be the FULL API path, e.g. /api/v1/devices/edge-poll
 *
 * The controller verifies the signature via DeviceAuthService.verifyRequestSignature
 * (timestamp window, nonce replay protection, device revocation, license state),
 * then dispatches. Entitlement (strategy-level) enforcement happens at delivery.
 */
@Controller('devices')
export class EdgePollController {
  constructor(
    private readonly edgePoll: EdgePollService,
    private readonly deviceAuth: DeviceAuthService,
  ) {}

  private async verify(req: any, headers: any, rawBody: any): Promise<string> {
    const deviceId = headers['x-device-id'];
    const timestamp = headers['x-device-timestamp'];
    const nonce = headers['x-device-nonce'];
    const signature = headers['x-device-signature'];
    if (!deviceId || !timestamp || !nonce || !signature) {
      throw new UnauthorizedException('device signature headers required');
    }

    // Raw body hash — use the RAW request body if available, else re-stringify JSON.
    let bodyStr = '';
    if (typeof rawBody === 'string') bodyStr = rawBody;
    else if (Buffer.isBuffer(req?.rawBody)) bodyStr = (req.rawBody as Buffer).toString('utf8');
    else if (req?.body && Object.keys(req.body).length) bodyStr = JSON.stringify(req.body);

    const crypto = await import('crypto');
    const bodyHash = crypto.createHash('sha256').update(bodyStr).digest('hex');

    // Path as the client signed it: /api/v1/devices/edge-poll.
    // req.route.path ALREADY includes the global prefix in Nest/Express
    // (e.g. "/api/v1/devices/edge-poll") — only prepend when it's absent.
    const routePath: string = req.route?.path || req.url?.split('?')[0] || '';
    const path = routePath.startsWith('/api/')
      ? routePath
      : `/api/v1/devices/${routePath.replace(/^\//, '')}`;

    const verdict = await this.deviceAuth.verifyRequestSignature({
      deviceId: String(deviceId),
      method: 'POST',
      path,
      bodyHash,
      timestamp: String(timestamp),
      nonce: String(nonce),
      signature: String(signature),
    });
    if (!verdict.valid) {
      throw new UnauthorizedException(`device signature invalid: ${verdict.reason}`);
    }
    return String(deviceId);
  }

  /**
   * POST /api/v1/devices/edge-poll
   * Returns pending EXECUTABLE signals for this device and marks them IN_FLIGHT.
   */
  @Post('edge-poll')
  async pollSignals(@Body() body: any, @Headers() headers: any, @Req() req: any) {
    const deviceId = await this.verify(req, headers, body);
    return this.edgePoll.poll(deviceId, body || {});
  }

  /**
   * POST /api/v1/devices/edge-ack
   * Device confirms execution result for a previously delivered queue item.
   */
  @Post('edge-ack')
  async edgeAck(@Body() body: any, @Headers() headers: any, @Req() req: any) {
    const deviceId = await this.verify(req, headers, body);
    return this.edgePoll.ack(deviceId, body || {});
  }

  /**
   * POST /api/v1/devices/edge-heartbeat
   * EA liveness + optional terminal/account metadata (admin dashboards).
   */
  @Post('edge-heartbeat')
  async edgeHeartbeat(@Body() body: any, @Headers() headers: any, @Req() req: any) {
    const deviceId = await this.verify(req, headers, body);
    const ip = req.headers['x-forwarded-for']?.split(',')[0]?.trim() || req.socket?.remoteAddress;
    return this.edgePoll.heartbeat(deviceId, body || {}, ip);
  }
  // v1.19.0 (Option B): POST /devices/edge-enqueue was REMOVED — the realtime
  // engine now writes licensing.edge_signal_queue directly (in-process DB),
  // so the HTTP proxy shim is dead. Keep the surface minimal.
}