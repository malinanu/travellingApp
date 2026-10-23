import type { Metadata } from 'next';

import LegalPage from '../../components/legal-page';

export const metadata: Metadata = {
  title: 'Terms of Service · Spotto Vendor Portal',
  description: 'The terms for listing a business on Spotto and using the vendor portal.',
};

export default function TermsPage() {
  return (
    <LegalPage
      title="Terms of Service"
      updated="23 October 2026"
      intro={
        <p>
          These terms cover listing your business on Spotto and using the Spotto Vendor Portal (the
          &ldquo;portal&rdquo;) to manage it. By creating a vendor account you agree to them on behalf of yourself
          and the business you register.
        </p>
      }
      sections={[
        {
          heading: 'Your account',
          body: (
            <>
              <p>
                You must give accurate details when you register and keep them up to date. You are responsible for
                everything done with your account, including by team members you invite, so keep your password
                private and turn on two-factor authentication if you can.
              </p>
              <p>Tell us straight away at <a href="mailto:partners@spotto.dev">partners@spotto.dev</a> if you think someone else has used your account.</p>
            </>
          ),
        },
        {
          heading: 'Listing review',
          body: (
            <p>
              New listings stay &ldquo;pending&rdquo; until our team reviews them. We may ask for proof that you own
              or are authorised to represent the business, such as a business registration certificate, and we may
              approve, reject, edit or remove a listing at our discretion, including when someone else has already
              listed the same place.
            </p>
          ),
        },
        {
          heading: 'Your content',
          body: (
            <>
              <p>
                You keep ownership of the business details, photos, menus, offers and other content you add. You
                give Spotto a non-exclusive, worldwide, royalty-free licence to host, display, adapt and promote that
                content in the Spotto apps, website and marketing for as long as it is on Spotto.
              </p>
              <p>
                Only upload content you have the right to use. It must be accurate and must not be misleading,
                unlawful, offensive or infringe anyone else&rsquo;s rights.
              </p>
            </>
          ),
        },
        {
          heading: 'Offers, redemptions and bookings',
          body: (
            <p>
              You decide which offers you publish and are responsible for honouring them, as described, for
              travellers who redeem them within their terms and limits. The same applies to bookings you accept.
              Spotto records claims, redemptions and bookings to run the service but is not a party to your
              transactions with travellers.
            </p>
          ),
        },
        {
          heading: 'Plans and fees',
          body: (
            <p>
              Listing on the Free plan costs nothing. Where paid plans or sponsored placements are offered, the
              portal shows their price and billing terms before you commit to them.
            </p>
          ),
        },
        {
          heading: 'Acceptable use',
          body: (
            <ul>
              <li>Don&rsquo;t post fake reviews, misleading offers or another business&rsquo;s listing as your own.</li>
              <li>Don&rsquo;t scrape, overload or try to break into the portal or other users&rsquo; accounts.</li>
              <li>Use traveller information you receive through Spotto only to serve that traveller.</li>
            </ul>
          ),
        },
        {
          heading: 'Suspension and closing your account',
          body: (
            <p>
              You can stop using Spotto at any time and ask us to remove your listing. We may suspend or close an
              account that breaks these terms or puts travellers at risk, and will tell you why unless the law
              prevents it.
            </p>
          ),
        },
        {
          heading: 'Disclaimers and liability',
          body: (
            <p>
              The portal is provided &ldquo;as is&rdquo;. We work to keep it available and accurate but can&rsquo;t
              guarantee it will be uninterrupted or error-free, or that listing will bring you a particular number
              of customers. To the extent the law allows, Spotto is not liable for indirect or consequential losses,
              such as lost profits, arising from your use of the portal.
            </p>
          ),
        },
        {
          heading: 'Changes and contact',
          body: (
            <p>
              We may update these terms; if a change is significant we will let you know in the portal or by email
              before it takes effect. These terms are governed by the laws of Sri Lanka. Questions? Email{' '}
              <a href="mailto:partners@spotto.dev">partners@spotto.dev</a>. See also our <a href="/privacy">Privacy Policy</a>.
            </p>
          ),
        },
      ]}
    />
  );
}
