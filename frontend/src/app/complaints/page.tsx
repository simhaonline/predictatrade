import type { Metadata } from "next";
import LegalLayout from "@/components/legal/legal-layout";
export const metadata: Metadata = { title: "Complaints Procedure | Predict-A-Trade" };
export default function ComplaintsPage() {
  return (
    <LegalLayout title="Complaints Procedure" lastUpdated="August 2026">
      <section><h2>1. How to Make a Complaint</h2><p>If you have a complaint about the Predict-A-Trade platform, please contact us at support@predictatrade.com with the subject line &quot;Complaint.&quot; Include your account email and a detailed description of the issue.</p></section>
      <section><h2>2. Acknowledgement</h2><p>We will acknowledge your complaint within 48 hours of receipt.</p></section>
      <section><h2>3. Investigation</h2><p>Your complaint will be investigated by our support team. We aim to resolve complaints within 5 business days. Complex cases may take longer, and we will keep you informed of progress.</p></section>
      <section><h2>4. Resolution</h2><p>We will provide a written response detailing our findings and any remedial action. If you are not satisfied with the resolution, you may escalate by replying to the response.</p></section>
      <section><h2>5. Escalation</h2><p>If the issue remains unresolved, it will be escalated to senior management for review. We are committed to fair and transparent complaint handling.</p></section>
      <section><h2>6. Contact Information</h2><p>Email: support@predictatrade.com</p></section>
    </LegalLayout>
  );
}
