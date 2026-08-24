import { ServiceUnavailableException, UnauthorizedException } from '@nestjs/common';
import { NowPaymentsService } from './nowpayments.service';

function makeService(pool: any) {
  const billingService = { generateInvoiceForSubscription: jest.fn().mockResolvedValue('inv-1') } as any;
  return new NowPaymentsService(pool, billingService);
}

describe('NowPaymentsService', () => {
  const ORIGINAL_ENV = process.env;

  beforeEach(() => {
    jest.resetModules();
    process.env = { ...ORIGINAL_ENV };
  });

  afterAll(() => {
    process.env = ORIGINAL_ENV;
  });

  describe('createInvoice fail-closed', () => {
    it('returns 503 payment_gateway_not_configured when NOWPAYMENTS_API_KEY is empty', async () => {
      delete process.env.NOWPAYMENTS_API_KEY;
      const pool = { query: jest.fn() };
      const service = makeService(pool);
      await expect(service.createInvoice('user-1', 'plan-1', 'MONTHLY')).rejects.toThrow(
        ServiceUnavailableException,
      );
      await expect(service.createInvoice('user-1', 'plan-1', 'MONTHLY')).rejects.toMatchObject({
        message: 'payment_gateway_not_configured',
      });
      // Must not touch the database or network when unconfigured.
      expect(pool.query).not.toHaveBeenCalled();
    });
  });

  describe('handleIPN security', () => {
    it('rejects when NOWPAYMENTS_IPN_SECRET is missing (fail closed)', async () => {
      delete process.env.NOWPAYMENTS_IPN_SECRET;
      const service = makeService({ query: jest.fn() });
      await expect(service.handleIPN({ payment_id: '1' }, 'sig')).rejects.toThrow(
        UnauthorizedException,
      );
    });

    it('rejects missing signature header', async () => {
      process.env.NOWPAYMENTS_IPN_SECRET = 's3cret';
      const service = makeService({ query: jest.fn() });
      await expect(service.handleIPN({ payment_id: '1' }, undefined)).rejects.toThrow(
        UnauthorizedException,
      );
    });

    it('rejects a tampered payload (signature mismatch)', async () => {
      process.env.NOWPAYMENTS_IPN_SECRET = 's3cret';
      const service = makeService({ query: jest.fn() });
      await expect(
        service.handleIPN({ payment_id: '9999', payment_status: 'finished' }, 'deadbeef'),
      ).rejects.toMatchObject({ message: 'invalid_ipn_signature' });
      // No state mutated on rejection.
    });

    it('accepts the documented canonical signature (sorted keys, key:value joined by |, HMAC-SHA512)', async () => {
      const secret = 's3cret';
      process.env.NOWPAYMENTS_IPN_SECRET = secret;
      const crypto = require('crypto');
      const body = { payment_status: 'confirmed', payment_id: '42', invoice_id: 'INV-7' };
      const sigString = Object.keys(body)
        .sort()
        .map((k) => `${k}:${(body as any)[k]}`)
        .join('|');
      const sig = crypto.createHmac('sha512', secret).update(sigString).digest('hex');
      const pool = {
        query: jest
          .fn()
          // payment_events insert -> duplicate (rowCount 0)
          .mockResolvedValueOnce({ rowCount: 0, rows: [] }),
      };
      const service = makeService(pool);
      const result = await service.handleIPN(body, sig);
      expect(result).toEqual({ received: true, duplicate: true });
      expect(pool.query).toHaveBeenCalledWith(
        expect.stringContaining("ON CONFLICT (provider, provider_event_id) DO NOTHING"),
        ['42:confirmed', 'confirmed', JSON.stringify(body)],
      );
    });
  });

  describe('handleIPN settlement', () => {
    function sign(body: Record<string, unknown>, secret: string): string {
      const crypto = require('crypto');
      const sigString = Object.keys(body)
        .sort()
        .map((k) => `${k}:${body[k]}`)
        .join('|');
      return crypto.createHmac('sha512', secret).update(sigString).digest('hex');
    }

    it('activates subscription and marks payment/invoice paid transactionally on finished', async () => {
      const secret = 's3cret';
      process.env.NOWPAYMENTS_IPN_SECRET = secret;
      const body = { payment_id: '77', payment_status: 'finished' };

      const clientQueries: any[] = [];
      const client = {
        query: jest.fn().mockImplementation((q: any, params?: any[]) => {
          clientQueries.push([q, params]);
          if (typeof q === 'string' && q.includes('SELECT * FROM billing.payments')) {
            return Promise.resolve({
              rows: [{ id: 'pay-uuid-1', status: 'PENDING', invoice_id: 'inv-uuid-1', subscription_id: 'sub-uuid-1' }],
            });
          }
          return Promise.resolve({ rows: [], rowCount: 1 });
        }),
        release: jest.fn(),
      };
      const pool = {
        connect: jest.fn().mockResolvedValue(client),
        query: jest.fn().mockResolvedValue({ rowCount: 1, rows: [{ id: 'evt-1' }] }),
      };
      const service = makeService(pool);
      const result = await service.handleIPN(body, sign(body, secret));

      expect(result).toEqual({ received: true, status: 'finished' });
      expect(client.query).toHaveBeenCalledWith('BEGIN');
      expect(client.query).toHaveBeenCalledWith(
        expect.stringContaining("SET status = 'COMPLETED'"),
        ['pay-uuid-1'],
      );
      expect(client.query).toHaveBeenCalledWith(
        expect.stringContaining("SET status = 'PAID'"),
        ['inv-uuid-1'],
      );
      expect(client.query).toHaveBeenCalledWith(
        expect.stringContaining("SET status = 'ACTIVE'"),
        ['sub-uuid-1'],
      );
      expect(client.query).toHaveBeenCalledWith(
        expect.stringContaining('audit.audit_events'),
        expect.arrayContaining(['pay-uuid-1']),
      );
      expect(client.query).toHaveBeenCalledWith('COMMIT');
      expect(client.release).toHaveBeenCalled();
    });

    it('ignores non-settling statuses without touching payments', async () => {
      const secret = 's3cret';
      process.env.NOWPAYMENTS_IPN_SECRET = secret;
      const body = { payment_id: '78', payment_status: 'waiting' };
      const client = { query: jest.fn(), release: jest.fn() };
      const pool = {
        connect: jest.fn(),
        query: jest.fn().mockResolvedValue({ rowCount: 1, rows: [{ id: 'evt-2' }] }),
      };
      const service = makeService(pool);
      const result = await service.handleIPN(body, sign(body, secret));
      expect(result).toEqual({ received: true, status: 'waiting' });
      expect(pool.connect).not.toHaveBeenCalled();
    });
  });
});
