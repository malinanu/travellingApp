import type { Metadata } from 'next';

import LegalPage from '../../components/legal-page';

export const metadata: Metadata = {
  title: 'Privacy Policy · Spotto Vendor Portal',
  description: 'What Spotto collects about vendor accounts and businesses, and how it is used.',
};

export default function PrivacyPage() {
  return (
    <LegalPage
      title="Privacy Policy"
      updated="23 October 2026"
      intro={
        <p>
          This policy explains what information Spotto collects when you create and use a vendor account in the
          Spotto Vendor Portal, how we use it, and the choices you have.
        </p>
      }
      sections={[
        {
          heading: 'What we collect',
          body: (
            <ul>
              <li><strong>Account details:</strong> your name, email address, phone number and password (stored only as a secure hash), plus your two-factor settings if you turn them on.</li>
              <li><strong>Business details:</strong> business name, category, address, map location, description, opening hours, contact details, photos, menu, offers and any documents you upload, such as a business registration certificate.</li>
              <li><strong>Activity:</strong> offers, redemptions, bookings and reviews linked to your business, and the team members you invite.</li>
              <li><strong>Technical data:</strong> sign-in sessions (device and browser type, IP address and times), used to keep your account secure and to limit abuse.</li>
            </ul>
          ),
        },
        {
          heading: 'How we use it',
          body: (
            <ul>
              <li>To create your account, review your listing and show approved business details to travellers in the Spotto apps.</li>
              <li>To verify that you represent the business, for example by reading the number on a registration certificate and checking it against the public company registry.</li>
              <li>To run offers, redemptions, bookings and reports for your business.</li>
              <li>To send you account emails, such as password resets and updates about your listing.</li>
              <li>To keep the service secure, prevent fraud and meet legal obligations.</li>
            </ul>
          ),
        },
        {
          heading: 'What travellers see',
          body: (
            <p>
              Once your listing is approved, travellers can see your business name, category, location, description,
              hours, photos, menu, offers and the public contact details you choose to add. Your account email,
              password and sign-in details are never shown to travellers.
            </p>
          ),
        },
        {
          heading: 'Service providers',
          body: (
            <p>
              We use trusted providers to run Spotto: hosting and file storage (such as Cloudflare), email delivery,
              and Google services for maps, place search, Google sign-in and automated reading of uploaded documents.
              They process information only on our behalf and under their own security obligations. We do not sell
              your personal information.
            </p>
          ),
        },
        {
          heading: 'How long we keep it',
          body: (
            <p>
              We keep your information while your account is active. If you close your account we delete or
              anonymise your personal information within a reasonable period, except where we need to keep records
              to resolve disputes, prevent fraud or comply with the law.
            </p>
          ),
        },
        {
          heading: 'Your choices',
          body: (
            <p>
              You can update most details yourself under Account and Profile in the portal. To get a copy of your
              information, correct something you can&rsquo;t edit, or delete your account, email{' '}
              <a href="mailto:partners@spotto.dev">partners@spotto.dev</a>.
            </p>
          ),
        },
        {
          heading: 'Security',
          body: (
            <p>
              Passwords are hashed, connections are encrypted, and you can sign out from Account at any time. No
              system is perfectly secure, so please use a strong, unique password and two-factor
              authentication.
            </p>
          ),
        },
        {
          heading: 'Changes and contact',
          body: (
            <p>
              We may update this policy and will note the date above; significant changes will be announced in the
              portal or by email. Questions? Email <a href="mailto:partners@spotto.dev">partners@spotto.dev</a>.
              See also our <a href="/terms">Terms of Service</a>.
            </p>
          ),
        },
      ]}
    />
  );
}
