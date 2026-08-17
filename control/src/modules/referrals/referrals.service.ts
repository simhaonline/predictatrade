import { Injectable, BadRequestException } from '@nestjs/common';

// SOW Section 69.7, 69.26: Referral relationship management with cycle prevention
@Injectable()
export class ReferralsService {
  // Build the five-level sponsor chain for a user
  // Returns [L1 userId, L2 userId, ... L5 userId]
  buildSponsorChain(
    userId: string,
    relationships: Map<string, string>, // child -> direct parent
  ): string[] {
    const chain: string[] = [];
    let current = userId;
    const visited = new Set<string>([userId]);

    for (let level = 1; level <= 5; level++) {
      const parent = relationships.get(current);
      if (!parent) break;

      // Cycle prevention (SOW Section 69.26)
      if (visited.has(parent)) {
        throw new BadRequestException('Circular referral detected');
      }
      visited.add(parent);
      chain.push(parent);
      current = parent;
    }

    return chain;
  }

  // SOW Section 69.26: Self-referral prevention
  validateNotSelfReferral(referrerId: string, referredId: string): void {
    if (referrerId === referredId) {
      throw new BadRequestException('Self-referral is not allowed');
    }
  }

  // SOW Section 69.16: First valid attribution wins
  validateAttribution(
    existingAttribution: { referrerId: string } | null,
    newReferrerId: string,
  ): { referrerId: string; isUpdate: boolean } {
    if (existingAttribution) {
      return { referrerId: existingAttribution.referrerId, isUpdate: false };
    }
    return { referrerId: newReferrerId, isUpdate: true };
  }
}
