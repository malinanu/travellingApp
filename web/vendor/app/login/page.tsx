'use client';

// Vendor auth — sign in, forgot/reset password, Google sign-in, and a
// 3-step "list your business" registration wizard (registration creates a
// pending vendor that an admin approves from the command center). A real
// TOTP step is inserted after credentials whenever the account has 2FA
// enabled (set up from Account → Two-factor authentication). Port of the
// VendorAuth mockup — Apple sign-in is intentionally not included, see
// SKIPPED.md.

import { API_URL, ApiError, isTwoFactorRequired, type AuthResponse, type TwoFactorRequiredResponse } from '@spotto/shared';
import {
  ArrowLeft, ArrowRight, Check, ChevronDown, Clock, Eye, EyeOff, Lock, Mail, MapPin,
  MailCheck, PartyPopper, Phone, QrCode, Search, Send, Store, Tag,
  TrendingUp, User as UserIcon, Users, LucideIcon,
} from 'lucide-react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useRef, useState } from 'react';

import AddressSearchField from '../../components/address-search-field';
import LocationPickerAuto, { preloadLocationPicker } from '../../components/location-picker-auto';
import { SpottoMark } from '../../components/shell';
import { api } from '../../lib/api';
import { VENDOR_SEGMENTS, categoriesBySegment, categoryById, categoryIconComponent } from '../../lib/business-categories';
import { loadGoogleMaps } from '../../lib/google-maps';

const REGISTER_STEPS = ['Find your business', 'Your account', 'Your business', 'A few details'];

// Default map-pin center for owners who skip the Step 1 business search
// entirely (placeLat/placeLng stay null in that case) — Colombo, matching
// the "every vendor is Sri Lanka-based" assumption used elsewhere in this
// wizard (see categoryFromGoogleTypes/townFromLegacyAddressComponents below).
const DEFAULT_PIN_LAT = 6.9271;
const DEFAULT_PIN_LNG = 79.8612;

// Best-effort Google place-type → Spotto category mapping, checked in
// array order (first match wins) — falls back to the wizard's existing
// 'coffee' default when nothing matches. The category grid in the next step
// stays fully visible either way, so a wrong guess is a one-tap fix.
const GOOGLE_TYPE_TO_CATEGORY: [string, string][] = [
  ['cafe', 'coffee'],
  ['restaurant', 'restaurants'],
  ['bar', 'bars'], ['night_club', 'nightlife'],
  ['bakery', 'takeout'],
  ['lodging', 'hotels'],
  ['spa', 'beauty-salons'],
  ['gym', 'gyms'],
  ['tourist_attraction', 'attractions'], ['travel_agency', 'attractions'],
  ['meal_takeaway', 'takeout'],
  ['art_gallery', 'art'],
  ['clothing_store', 'apparel'], ['store', 'shopping-centers'],
];

function categoryFromGoogleTypes(types: string[] | undefined): string {
  for (const [googleType, spottoCategory] of GOOGLE_TYPE_TO_CATEGORY) {
    if (types?.includes(googleType)) return spottoCategory;
  }
  return 'coffee';
}

// Sri Lankan addresses don't always have a `locality` (small towns often
// only show up as a sublocality or the district-level admin area) — check
// in order of specificity and take the first that's present.
const TOWN_COMPONENT_PRIORITY = ['locality', 'sublocality', 'administrative_area_level_2', 'administrative_area_level_1'];

// Uses the classic google.maps.places.Autocomplete widget (bound to a
// plain <input>) rather than the newer PlaceAutocompleteElement — that
// element manages its own internal shadow-DOM input with no reliable way
// to style it to match this wizard (it rendered as an unstyled black box
// in practice), and no way to pre-fill it either. Same deliberate choice
// already made in components/address-search-field.tsx — see its own doc
// comment. GeocoderAddressComponent's fields (long_name/short_name) are
// the classic-API shape, hence "Legacy" in these names.
function townFromLegacyAddressComponents(components: google.maps.GeocoderAddressComponent[] | undefined): string {
  for (const type of TOWN_COMPONENT_PRIORITY) {
    const match = components?.find((c) => c.types.includes(type));
    if (match) return match.long_name;
  }
  return '';
}

// The 'country' component's short_name is the ISO 3166-1 alpha-2 code
// (e.g. "LK", "US") — exactly what the backend's Vendor.Country expects,
// no guessing needed since it comes straight from Google.
function countryFromLegacyAddressComponents(components: google.maps.GeocoderAddressComponent[] | undefined): string {
  return components?.find((c) => c.types.includes('country'))?.short_name ?? '';
}

// Google numbers days Sunday=0..Saturday=6 in PlaceOpeningHoursPeriod —
// same convention server/internal/http/vendor.go's weekdayKeys already uses.
const GOOGLE_WEEKDAYS = ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday'];

function formatLegacyTime(time: string): string {
  return `${time.slice(0, 2)}:${time.slice(2, 4)}`;
}

function hoursFromLegacyPeriods(
  periods: google.maps.places.PlaceOpeningHoursPeriod[] | undefined,
): Record<string, { open: string; close: string }> | null {
  if (!periods || periods.length === 0) return null;
  const hours: Record<string, { open: string; close: string }> = {};
  for (const p of periods) {
    if (!p.open) continue;
    const open = formatLegacyTime(p.open.time);
    const close = p.close ? formatLegacyTime(p.close.time) : open;
    hours[GOOGLE_WEEKDAYS[p.open.day]] = { open, close };
  }
  return Object.keys(hours).length > 0 ? hours : null;
}

// One email is one Spotto account with one role, so an email already used
// in the traveller app can't also open (or register in) this portal. Say
// that, and how to get a second address, instead of bouncing the owner
// between "sign in instead" and "this portal is for business accounts".
const EMAIL_TAKEN_MESSAGE = 'This email already has a Spotto account. If it’s your business account, sign in. If you use it in the Spotto traveller app, list your business with a different email — with Gmail, you+business@gmail.com still reaches your inbox.';
const NOT_A_VENDOR_MESSAGE = 'This email belongs to a Spotto account that isn’t a business account, like the one you use in the Spotto app. To list a business, choose “List your business” below and use a different email — with Gmail, you+business@gmail.com still reaches your inbox.';

// The +94 is shown as a fixed prefix, but owners often type their number
// the local way (076…) or paste it with the country code — keep only the
// subscriber digits so the stored number has exactly one +94.
function sriLankanPhone(input: string): string {
  let digits = input.replace(/\D/g, '');
  if (digits.startsWith('94') && digits.length > 9) digits = digits.slice(2);
  if (digits.startsWith('0')) digits = digits.slice(1);
  return digits ? `+94 ${digits}` : '';
}

function GoogleG({ size = 18 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 48 48" aria-hidden="true">
      <path fill="#4285F4" d="M45.1 24.5c0-1.6-.1-3.1-.4-4.5H24v8.5h11.8c-.5 2.7-2 5-4.4 6.6v5.5h7.1c4.1-3.8 6.6-9.4 6.6-16.1z" />
      <path fill="#34A853" d="M24 46c5.9 0 10.9-2 14.5-5.3l-7.1-5.5c-2 1.3-4.5 2.1-7.4 2.1-5.7 0-10.5-3.8-12.2-9h-7.3v5.7C8.1 41.1 15.4 46 24 46z" />
      <path fill="#FBBC05" d="M11.8 28.3c-.4-1.3-.7-2.7-.7-4.3s.3-3 .7-4.3v-5.7H4.5C3 17.1 2.1 20.4 2.1 24s.9 6.9 2.4 9.7l7.3-5.4z" />
      <path fill="#EA4335" d="M24 10.7c3.2 0 6.1 1.1 8.4 3.3l6.3-6.3C34.9 4.1 29.9 2 24 2 15.4 2 8.1 6.9 4.5 14.3l7.3 5.7c1.7-5.2 6.5-9.3 12.2-9.3z" />
    </svg>
  );
}

function Field({
  label, type = 'text', value, onChange, icon: Icon, reveal, onReveal, autoFocus, placeholder,
}: {
  label: string; type?: string; value: string; onChange: (v: string) => void; icon?: LucideIcon;
  reveal?: boolean; onReveal?: () => void; autoFocus?: boolean; placeholder?: string;
}) {
  return (
    <label className="block">
      <span className="block text-[12.5px] font-extrabold mb-1.5 text-ink-soft">{label}</span>
      <div className="flex items-center gap-2.5 rounded-2xl px-3.5 h-[50px] bg-white border-[1.5px] border-line">
        {Icon && <Icon size={18} className="text-muted shrink-0" />}
        <input type={type} value={value} onChange={(e) => onChange(e.target.value)} autoFocus={autoFocus} placeholder={placeholder}
          className="flex-1 bg-transparent outline-none text-[15px] font-bold w-full text-ink" />
        {onReveal && (
          <button type="button" onClick={onReveal} className="text-muted shrink-0">
            {reveal ? <EyeOff size={18} /> : <Eye size={18} />}
          </button>
        )}
      </div>
    </label>
  );
}

function Primary({ children, onClick, disabled, icon: Icon = ArrowRight, type }: {
  children: React.ReactNode; onClick?: () => void; disabled?: boolean; icon?: LucideIcon | null; type?: 'submit' | 'button';
}) {
  return (
    <button type={type ?? 'button'} onClick={onClick} disabled={disabled}
      className="btn-accent w-full rounded-2xl py-3.5 text-[15.5px] flex items-center justify-center gap-2 whitespace-nowrap transition-transform active:scale-[0.98] disabled:opacity-60"
      style={{ boxShadow: '0 14px 28px -12px var(--accent-shadow, rgba(11,111,134,0.35))' }}>
      {children}
      {Icon && <Icon size={18} strokeWidth={2.5} />}
    </button>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <LoginPageInner />
    </Suspense>
  );
}

function LoginPageInner() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [mode, setMode] = useState<'login' | 'register' | 'forgot'>('login');
  const [registerStep, setRegisterStep] = useState(0);
  const [otpMode, setOtpMode] = useState(false);
  const [twoFactorToken, setTwoFactorToken] = useState<string | null>(null);
  const [otp, setOtp] = useState(['', '', '', '', '', '']);
  const otpRefs = useRef<(HTMLInputElement | null)[]>([]);

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [remember, setRemember] = useState(true);
  const [agreeTos, setAgreeTos] = useState(false);

  const [name, setName] = useState('');
  const [businessName, setBusinessName] = useState('');
  const [category, setCategory] = useState('coffee');
  // Which category segment is expanded on the "Your business" step. null
  // follows the chosen category (including one auto-filled from the
  // Google business search); '' means the owner collapsed them all.
  const [openSegment, setOpenSegment] = useState<string | null>(null);
  const shownSegment = openSegment ?? categoryById(category)?.segment ?? VENDOR_SEGMENTS[0].id;
  const [phone, setPhone] = useState('');
  const [town, setTown] = useState('');
  const [description, setDescription] = useState('');

  // Populated by the "Find your business" step's Google Places search —
  // all optional, sent with the registration request only when present.
  const [placeLat, setPlaceLat] = useState<number | null>(null);
  const [placeLng, setPlaceLng] = useState<number | null>(null);
  // Bumped whenever Step 4's Address search resolves a new location, so
  // the pin picker pans there — see LocationPickerAuto's recenterToken prop.
  const [pinSeq, setPinSeq] = useState(0);
  const [foundPlace, setFoundPlace] = useState<{ name: string; address: string } | null>(null);
  const [businessContact, setBusinessContact] = useState<{ phone: string; website: string } | null>(null);
  const [businessHours, setBusinessHours] = useState<Record<string, { open: string; close: string }> | null>(null);
  const [businessCountry, setBusinessCountry] = useState('');
  const [googlePlaceId, setGooglePlaceId] = useState('');
  const businessSearchInputRef = useRef<HTMLInputElement>(null);
  const mapDivRef = useRef<HTMLDivElement>(null);

  const [forgotEmail, setForgotEmail] = useState('');
  const [forgotSent, setForgotSent] = useState(false);
  const [resetToken, setResetToken] = useState<string | null>(null);
  const [newPassword, setNewPassword] = useState('');
  const [resetDone, setResetDone] = useState(false);

  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  // One-time handling of whatever landed us here via a redirect: an OAuth
  // exchange code, a password-reset link, an OAuth "no account, please
  // register" bounce, or an OAuth error.
  useEffect(() => {
    const authCode = searchParams.get('authCode');
    const rToken = searchParams.get('resetToken');
    const oauthMode = searchParams.get('mode');
    const oauthError = searchParams.get('error');

    if (authCode) {
      setBusy(true);
      api.post<AuthResponse | TwoFactorRequiredResponse>('/auth/exchange', { code: authCode })
        .then((res) => {
          if (isTwoFactorRequired(res)) {
            setTwoFactorToken(res.twoFactorToken);
            setOtpMode(true);
            setTimeout(() => otpRefs.current[0]?.focus(), 50);
          } else {
            finishSession(res, true);
          }
        })
        .catch((err) => setError(err instanceof Error ? err.message : 'Google sign-in failed'))
        .finally(() => { setBusy(false); router.replace('/login'); });
      return;
    }
    if (rToken) {
      setResetToken(rToken);
      setMode('forgot');
      router.replace('/login');
      return;
    }
    if (oauthMode === 'register') {
      setMode('register');
      setRegisterStep(0);
      setEmail(searchParams.get('email') ?? '');
      setName(searchParams.get('name') ?? '');
      router.replace('/login');
      return;
    }
    if (oauthError === 'not_a_vendor_account') {
      setError(NOT_A_VENDOR_MESSAGE);
      router.replace('/login');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // The map only renders on the last step; fetch its code as soon as the
  // wizard opens so a deploy part-way through doesn't strand the owner.
  useEffect(() => {
    if (mode === 'register') preloadLocationPicker();
  }, [mode]);

  // Loads Google Places Autocomplete for the "Find your business" step
  // (registerStep 0) — only while that step is actually visible, so coming
  // Back to it re-attaches the search.
  useEffect(() => {
    if (mode !== 'register' || registerStep !== 0) return;
    let cancelled = false;
    let autocomplete: google.maps.places.Autocomplete | null = null;
    loadGoogleMaps().then(async (g) => {
      if (cancelled || !businessSearchInputRef.current) return;

      // Re-entering this step (e.g. via Back) after already finding a
      // place: restore the map preview (the found-place card below the
      // search box already shows the name/address).
      if (foundPlace && placeLat != null && placeLng != null && mapDivRef.current) {
        const map = new g.maps.Map(mapDivRef.current, {
          center: { lat: placeLat, lng: placeLng }, zoom: 15,
          disableDefaultUI: true, gestureHandling: 'none',
        });
        new g.maps.Marker({ position: { lat: placeLat, lng: placeLng }, map });
      }

      const { Autocomplete } = await g.maps.importLibrary('places') as google.maps.PlacesLibrary;
      autocomplete = new Autocomplete(businessSearchInputRef.current, {
        fields: [
          'place_id', 'name', 'formatted_address', 'address_components',
          'international_phone_number', 'website', 'geometry', 'opening_hours', 'types',
        ],
        componentRestrictions: { country: 'lk' },
      });
      autocomplete.addListener('place_changed', () => {
        const place = autocomplete!.getPlace();
        const loc = place.geometry?.location;
        if (!loc) return;
        const lat = loc.lat();
        const lng = loc.lng();

        setPlaceLat(lat);
        setPlaceLng(lng);
        setFoundPlace({ name: place.name ?? '', address: place.formatted_address ?? '' });
        setBusinessName(place.name ?? '');
        const foundTown = townFromLegacyAddressComponents(place.address_components);
        if (foundTown) setTown(foundTown);
        setBusinessCountry(countryFromLegacyAddressComponents(place.address_components));
        setCategory(categoryFromGoogleTypes(place.types));
        setBusinessContact(
          place.international_phone_number || place.website
            ? { phone: place.international_phone_number ?? '', website: place.website ?? '' }
            : null,
        );
        setBusinessHours(hoursFromLegacyPeriods(place.opening_hours?.periods));
        setGooglePlaceId(place.place_id ?? '');

        if (mapDivRef.current) {
          const map = new g.maps.Map(mapDivRef.current, {
            center: loc, zoom: 15, disableDefaultUI: true, gestureHandling: 'none',
          });
          new g.maps.Marker({ position: loc, map });
        }
      });
    }).catch((err) => {
      // Google couldn't load (blocked, offline, bad key): the owner can
      // still skip search and fill the details in by hand.
      console.warn('Business search unavailable, Google Maps failed to load:', err);
    });
    return () => {
      if (autocomplete) google.maps.event.clearInstanceListeners(autocomplete);
      cancelled = true;
    };
    // Deliberately excludes foundPlace/placeLat/placeLng etc — those are
    // set BY this effect's own listener, so including them would re-fire
    // this setup (re-attaching a fresh Autocomplete instance) immediately
    // after every place selection.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, registerStep]);

  // "Enter your business details manually instead" after picking a place:
  // drop what that place filled in behind the scenes (its Google listing,
  // pin, phone/website, hours), so the registration doesn't quietly carry
  // a listing the owner backed out of. Visible fields (name, town,
  // category) stay, since the owner can see and edit those.
  const clearFoundPlace = () => {
    setFoundPlace(null);
    setPlaceLat(null);
    setPlaceLng(null);
    setBusinessContact(null);
    setBusinessHours(null);
    setBusinessCountry('');
    setGooglePlaceId('');
  };

  const finishSession = (res: AuthResponse, rememberOverride?: boolean) => {
    if (res.user.role !== 'vendor') {
      api.setToken(null);
      api.logout(res.refreshToken);
      setError(NOT_A_VENDOR_MESSAGE);
      return;
    }
    if (!res.user.vendorId) {
      // A vendor account whose business was never created or was deleted:
      // every vendor page would just fail, so don't sign it in.
      api.setToken(null);
      api.logout(res.refreshToken);
      setError('There’s no business on this account. Register your business below, or contact partners@spotto.dev.');
      return;
    }
    api.setToken(res.token, rememberOverride ?? remember, res.refreshToken);
    router.replace('/dashboard');
  };

  const submitLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await api.post<AuthResponse | TwoFactorRequiredResponse>('/auth/login', { email, password });
      if (isTwoFactorRequired(res)) {
        setTwoFactorToken(res.twoFactorToken);
        setOtpMode(true);
        setTimeout(() => otpRefs.current[0]?.focus(), 50);
        return;
      }
      finishSession(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong');
    } finally {
      setBusy(false);
    }
  };

  // Set when the account step's email turns out to be taken, so the error
  // can offer "Sign in instead" rather than a dead end.
  const [emailTaken, setEmailTaken] = useState(false);

  // Account step: check the email and password with the server before
  // moving on, so "email already registered" or a weak password shows up
  // here instead of after the owner has filled in the whole wizard.
  const continueFromAccount = async () => {
    setBusy(true);
    setError(null);
    setEmailTaken(false);
    try {
      await api.post('/auth/register/check', { email, password });
      setRegisterStep(2);
    } catch (err) {
      // A server older than this check: carry on, and sign-up itself will
      // still catch a taken email.
      if (err instanceof ApiError && err.status === 404) {
        setRegisterStep(2);
        return;
      }
      setEmailTaken(err instanceof ApiError && err.status === 409);
      setError(err instanceof ApiError && err.status === 409
        ? EMAIL_TAKEN_MESSAGE
        : err instanceof Error ? err.message : 'Something went wrong');
    } finally {
      setBusy(false);
    }
  };

  const submitRegister = async () => {
    setBusy(true);
    setError(null);
    try {
      const res = await api.post<AuthResponse | TwoFactorRequiredResponse>('/auth/register', {
        email, password, name, role: 'vendor', businessName, category,
        phone: sriLankanPhone(phone), town, description,
        ...(placeLat != null && placeLng != null ? { lat: placeLat, lng: placeLng } : {}),
        ...(businessContact ? { contact: businessContact } : {}),
        ...(businessHours ? { hours: businessHours } : {}),
        ...(businessCountry ? { businessCountry } : {}),
        ...(googlePlaceId ? { googlePlaceId } : {}),
      });
      if (isTwoFactorRequired(res)) {
        setTwoFactorToken(res.twoFactorToken);
        setOtpMode(true);
        setTimeout(() => otpRefs.current[0]?.focus(), 50);
        return;
      }
      // Hold onto the session but don't navigate yet — show the
      // "Application received" confirmation first (registerStep 4).
      api.setToken(res.token, remember, res.refreshToken);
      setRegisterStep(4);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        // Taken since the account step (or that check was skipped): send
        // the owner back to the step where the email can be changed.
        setRegisterStep(1);
        setEmailTaken(true);
        setError(EMAIL_TAKEN_MESSAGE);
      } else {
        setError(err instanceof Error ? err.message : 'Something went wrong');
      }
      api.setToken(null);
    } finally {
      setBusy(false);
    }
  };

  const onOtpChange = (i: number, v: string) => {
    const digit = v.replace(/\D/g, '').slice(-1);
    const next = [...otp];
    next[i] = digit;
    setOtp(next);
    if (digit && i < 5) otpRefs.current[i + 1]?.focus();
  };

  const submitOtp = async () => {
    if (otp.join('').length !== 6 || !twoFactorToken) return;
    setBusy(true);
    setError(null);
    try {
      const res = await api.post<AuthResponse>('/auth/2fa/login-verify', { twoFactorToken, code: otp.join('') });
      finishSession(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invalid code');
      setOtp(['', '', '', '', '', '']);
      otpRefs.current[0]?.focus();
    } finally {
      setBusy(false);
    }
  };

  const submitForgot = async () => {
    setBusy(true);
    setError(null);
    try {
      await api.post('/auth/forgot-password', { email: forgotEmail });
      setForgotSent(true);
    } catch {
      // handleForgotPassword always 200s — this only fails on network error.
      setError('Something went wrong — try again in a moment.');
    } finally {
      setBusy(false);
    }
  };

  const submitReset = async () => {
    setBusy(true);
    setError(null);
    try {
      await api.post('/auth/reset-password', { token: resetToken, newPassword });
      setResetDone(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not reset password');
    } finally {
      setBusy(false);
    }
  };

  if (otpMode) {
    return (
      <AuthShell>
        <h1 className="font-display font-extrabold leading-tight text-[28px]">Two-factor verification</h1>
        <p className="text-[14px] text-muted font-semibold mt-1.5 mb-6">Enter the 6-digit code from your authenticator app.</p>
        <div className="flex gap-2 mb-4">
          {otp.map((d, i) => (
            <input key={i} ref={(el) => { otpRefs.current[i] = el; }} value={d} onChange={(e) => onOtpChange(i, e.target.value)}
              inputMode="numeric" maxLength={1} className="input text-center text-xl font-extrabold w-12 h-14" />
          ))}
        </div>
        {error && <p className="text-[13.5px] font-bold text-coral mb-3">{error}</p>}
        <Primary onClick={submitOtp} disabled={otp.join('').length !== 6 || busy} icon={busy ? null : ArrowRight}>
          {busy ? 'Verifying…' : 'Verify & continue'}
        </Primary>
      </AuthShell>
    );
  }

  if (mode === 'forgot') {
    if (resetToken) {
      return (
        <AuthShell>
          <h1 className="font-display font-extrabold leading-tight text-[28px]">Set a new password</h1>
          <p className="text-[14px] text-muted font-semibold mt-1.5 mb-6">Choose a new password for your account.</p>
          {resetDone ? (
            <div className="rounded-2xl bg-accent-tint p-4 flex items-start gap-3 mb-5">
              <MailCheck size={20} className="text-accent shrink-0" />
              <p className="text-[13.5px] font-bold leading-snug">Password updated — sign in with your new password.</p>
            </div>
          ) : (
            <div className="mb-5">
              <Field label="New password" type="password" value={newPassword} onChange={setNewPassword} icon={Lock} placeholder="Min 10 characters" autoFocus />
            </div>
          )}
          {error && <p className="text-[13.5px] font-bold text-coral mb-3">{error}</p>}
          {resetDone ? (
            <Primary onClick={() => { setMode('login'); setResetToken(null); setResetDone(false); }}>Back to sign in</Primary>
          ) : (
            <Primary onClick={submitReset} disabled={newPassword.length < 10 || busy} icon={Send}>
              {busy ? 'Saving…' : 'Save new password'}
            </Primary>
          )}
        </AuthShell>
      );
    }
    return (
      <AuthShell>
        <h1 className="font-display font-extrabold leading-tight text-[28px]">Reset password</h1>
        <p className="text-[14px] text-muted font-semibold mt-1.5 mb-6">We&rsquo;ll email you a secure reset link.</p>
        {forgotSent ? (
          <div className="rounded-2xl bg-accent-tint p-4 flex items-start gap-3 mb-5">
            <MailCheck size={20} className="text-accent shrink-0" />
            <p className="text-[13.5px] font-bold leading-snug">Check your inbox — the reset link is on its way.</p>
          </div>
        ) : (
          <div className="mb-5">
            <Field label="Email" type="email" value={forgotEmail} onChange={setForgotEmail} icon={Mail} placeholder="you@yourbusiness.lk" autoFocus />
          </div>
        )}
        {error && <p className="text-[13.5px] font-bold text-coral mb-3">{error}</p>}
        {!forgotSent && (
          <Primary onClick={submitForgot} disabled={!forgotEmail || busy} icon={Send}>
            {busy ? 'Sending…' : 'Send reset link'}
          </Primary>
        )}
        <button onClick={() => { setForgotSent(false); setMode('login'); }} className="w-full text-center text-[13.5px] font-extrabold mt-4 text-muted">
          ← Back to sign in
        </button>
      </AuthShell>
    );
  }

  if (mode === 'register') {
    if (registerStep >= 4) {
      return (
        <AuthShell wide>
          <div className="text-center">
            <div className="w-16 h-16 rounded-full bg-accent-tint flex items-center justify-center mx-auto mb-5">
              <PartyPopper size={30} className="text-accent" />
            </div>
            <h1 className="font-display font-extrabold leading-tight text-[26px]">Application received!</h1>
            <p className="text-[14.5px] text-muted font-semibold mt-2.5 mb-6 leading-relaxed">
              Our team reviews new listings within 24 hours. Meanwhile, take a look around your dashboard.
            </p>
            <div className="rounded-2xl bg-sand border-[1.5px] border-line p-4 mb-6 text-left flex items-center gap-3">
              <Clock size={20} className="text-accent shrink-0" />
              <div>
                <p className="text-[13.5px] font-extrabold">Status: Pending review</p>
                <p className="text-[12.5px] text-muted font-semibold">We&rsquo;ll email you once you&rsquo;re verified.</p>
              </div>
            </div>
            <Primary onClick={() => router.replace('/dashboard')}>Explore the dashboard</Primary>
          </div>
        </AuthShell>
      );
    }

    return (
      <AuthShell wide>
        <div className="flex items-center gap-2 mb-1">
          <span className="text-[11.5px] font-extrabold uppercase tracking-wider text-accent">Step {registerStep + 1} of 4</span>
        </div>
        <div className="flex gap-1.5 mb-5">
          {[0, 1, 2, 3].map((i) => (
            <div key={i} className={`h-1.5 rounded-full flex-1 transition-colors ${i <= registerStep ? 'bg-accent' : 'bg-[rgba(15,61,58,0.12)]'}`} />
          ))}
        </div>
        <h1 className="font-display font-extrabold leading-tight text-[25px] mb-5">{REGISTER_STEPS[registerStep]}</h1>

        {registerStep === 0 && (
          <div className="space-y-4">
            <label className="block">
              <span className="block text-[12.5px] font-extrabold mb-1.5 text-ink-soft">Search for your business</span>
              <div className="flex items-center gap-2.5 rounded-2xl px-3.5 h-[50px] bg-white border-[1.5px] border-line">
                <Search size={18} className="text-muted shrink-0" />
                <input ref={businessSearchInputRef} autoFocus placeholder="e.g. Driftwood Coffee & Co-Work"
                  className="flex-1 bg-transparent outline-none text-[15px] font-bold w-full text-ink" />
              </div>
            </label>

            {foundPlace && (
              <div className="rounded-2xl border-[1.5px] border-line overflow-hidden bg-white">
                <div ref={mapDivRef} className="h-[130px] w-full bg-sand" />
                <div className="p-3.5 flex items-start gap-2.5">
                  <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center shrink-0"><MapPin size={17} /></span>
                  <div className="min-w-0">
                    <p className="font-extrabold text-[14px] truncate">{foundPlace.name}</p>
                    <p className="text-[12.5px] text-muted font-semibold truncate">{foundPlace.address}</p>
                  </div>
                </div>
              </div>
            )}

            <p className="text-[12.5px] text-muted font-semibold text-center">
              We&rsquo;ll use this to pre-fill your business name, category, town and hours — you can adjust anything next.
            </p>

            <button type="button" onClick={() => { clearFoundPlace(); setRegisterStep(1); }}
              className="w-full text-center text-[13px] font-extrabold text-muted">
              Enter your business details manually instead
            </button>
          </div>
        )}

        {registerStep === 1 && (
          <div className="space-y-3.5">
            <Field label="Owner / contact name" value={name} onChange={setName} icon={UserIcon} placeholder="e.g. Tharindu Perera" autoFocus />
            <Field label="Email" type="email" value={email} onChange={setEmail} icon={Mail} placeholder="you@yourbusiness.lk" />
            <Field label="Password" type={showPassword ? 'text' : 'password'} value={password} onChange={setPassword} icon={Lock}
              reveal={showPassword} onReveal={() => setShowPassword((s) => !s)} placeholder="Create a password (min 10 characters)" />
          </div>
        )}

        {registerStep === 2 && (
          <div className="space-y-4">
            <Field label="Business name" value={businessName} onChange={setBusinessName} icon={Store} placeholder="e.g. Driftwood Coffee & Co-Work" autoFocus />
            <div>
              <span className="block text-[12.5px] font-extrabold mb-2 text-ink-soft">Category</span>
              <div className="space-y-2.5">
                {VENDOR_SEGMENTS.map((segment) => {
                  const items = categoriesBySegment(segment.id);
                  const open = shownSegment === segment.id;
                  const picked = items.find((t) => t.id === category);
                  return (
                    <div key={segment.id} className="rounded-2xl bg-white border-[1.5px] border-line overflow-hidden">
                      <button type="button" aria-expanded={open}
                        onClick={() => setOpenSegment(open ? '' : segment.id)}
                        className="w-full flex items-center gap-2 px-3.5 py-3 text-left">
                        <span className="font-extrabold text-[13px] text-ink">{segment.label}</span>
                        <span className="text-[11.5px] font-bold text-muted">{items.length}</span>
                        {picked && !open && (
                          <span className="ml-1 rounded-full px-2 py-0.5 text-[11px] font-extrabold bg-accent-tint text-accent truncate">
                            {picked.label}
                          </span>
                        )}
                        <ChevronDown size={17}
                          className={`ml-auto text-muted shrink-0 transition-transform ${open ? 'rotate-180' : ''}`} />
                      </button>
                      {open && (
                        <div className="grid grid-cols-2 gap-2.5 px-3 pb-3">
                          {items.map((t) => {
                            const Icon = categoryIconComponent(t);
                            const on = category === t.id;
                            return (
                              <button key={t.id} type="button" onClick={() => setCategory(t.id)}
                                className={`flex items-center gap-2.5 rounded-2xl px-3 py-3 text-left transition-all active:scale-[0.98] border-[1.5px] ${
                                  on ? 'bg-accent border-accent text-white' : 'bg-white border-line text-ink'
                                }`}>
                                <Icon size={18} />
                                <span className="font-extrabold text-[12.5px] leading-tight">{t.label}</span>
                              </button>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        )}

        {registerStep === 3 && (
          <div className="space-y-3.5">
            <div className="flex gap-2.5 items-end">
              <div className="shrink-0">
                <span className="block text-[12.5px] font-extrabold mb-1.5 text-transparent select-none" aria-hidden="true">LK</span>
                <div className="flex items-center gap-1.5 rounded-2xl px-3 h-[50px] bg-white border-[1.5px] border-line">
                  <span className="text-[15px]">🇱🇰</span><span className="font-extrabold text-[15px]">+94</span>
                </div>
              </div>
              <div className="flex-1">
                <Field label="Phone number" type="tel" value={phone} onChange={setPhone} icon={Phone} placeholder="76 123 4891" />
              </div>
            </div>
            <AddressSearchField
              value={town}
              onChange={setTown}
              onLocationSelect={(lat, lng) => { setPlaceLat(lat); setPlaceLng(lng); setPinSeq((s) => s + 1); }}
              onPlaceDetails={({ phone, website, hours }) => {
                // Category isn't wired here: Step 2 already has the owner
                // pick one (defaulting to 'coffee'), so there's no reliable
                // "still empty" signal the way there is for businessContact/
                // businessHours, which start out as null.
                if ((phone || website) && !businessContact) {
                  setBusinessContact({ phone: phone ?? '', website: website ?? '' });
                }
                if (hours && !businessHours) setBusinessHours(hours);
              }}
              placeholder="Search for your address…"
            />
            <div>
              <span className="block text-[12.5px] font-extrabold mb-1.5 text-ink-soft">Pin your exact location</span>
              <p className="text-[12px] text-muted font-semibold mb-1.5">
                Click anywhere on the map to move your pin — this is where travelers get directions to.
              </p>
              <div className="h-[180px] rounded-2xl overflow-hidden border-[1.5px] border-line">
                <LocationPickerAuto
                  lat={placeLat ?? DEFAULT_PIN_LAT}
                  lng={placeLng ?? DEFAULT_PIN_LNG}
                  recenterToken={pinSeq}
                  onChange={(lat, lng) => { setPlaceLat(lat); setPlaceLng(lng); }}
                />
              </div>
              {/* The map shows a placeholder pin before one is set; saving
                  that would put the business somewhere it isn't, so ask
                  for a real location first. */}
              {placeLat == null && (
                <p className="text-[12px] font-bold text-coral mt-1.5">
                  Set your location to continue: search your address above, or click the map.
                </p>
              )}
            </div>
            <label className="block">
              <span className="block text-[12.5px] font-extrabold mb-1.5 text-ink-soft">Tell travelers about your place</span>
              <textarea rows={3} value={description} onChange={(e) => setDescription(e.target.value)}
                placeholder="One or two lines on what makes it special…"
                className="w-full rounded-2xl px-3.5 py-3 bg-white border-[1.5px] border-line outline-none text-[15px] font-bold resize-none text-ink" />
            </label>
            <label className="flex items-start gap-2.5 mt-4 cursor-pointer">
              <input type="checkbox" checked={agreeTos} onChange={(e) => setAgreeTos(e.target.checked)} className="mt-0.5 rounded border-line accent-accent w-4 h-4" />
              <span className="text-[13px] text-muted font-semibold">
                I agree to the <a href="/terms" target="_blank" rel="noopener noreferrer" className="text-accent underline font-extrabold">Terms of Service</a> and <a href="/privacy" target="_blank" rel="noopener noreferrer" className="text-accent underline font-extrabold">Privacy Policy</a>.
              </span>
            </label>
          </div>
        )}

        {error && (
          <p className="text-[13.5px] font-bold text-coral mt-4">
            {error}
            {emailTaken && (
              <>
                {' '}
                <button type="button" className="text-accent underline font-extrabold"
                  onClick={() => { setMode('login'); setError(null); setEmailTaken(false); }}>
                  Sign in
                </button>
              </>
            )}
          </p>
        )}

        <div className="flex gap-3 mt-6">
          {registerStep > 0 && (
            <button type="button" onClick={() => { setError(null); setEmailTaken(false); setRegisterStep((s) => s - 1); }}
              className="rounded-2xl px-5 py-3.5 font-extrabold text-[15px] bg-white border-[1.5px] border-line active:scale-[0.98]">
              <ArrowLeft size={18} />
            </button>
          )}
          <div className="flex-1">
            <Primary
              disabled={busy || (registerStep === 1 && (!name || !email || password.length < 10)) || (registerStep === 2 && !businessName) || (registerStep === 3 && (!agreeTos || placeLat == null))}
              icon={busy ? null : registerStep === 3 ? Check : ArrowRight}
              onClick={() => {
                if (registerStep === 3) return submitRegister();
                if (registerStep === 1) return continueFromAccount();
                setError(null);
                setRegisterStep((s) => s + 1);
              }}
            >
              {busy ? (registerStep === 1 ? 'Checking…' : 'Submitting…') : registerStep === 3 ? 'Submit application' : 'Continue'}
            </Primary>
          </div>
        </div>

        <p className="text-center text-[13.5px] font-bold mt-5 text-muted">
          Already a partner?{' '}
          <button type="button" onClick={() => { setMode('login'); setError(null); }} className="font-extrabold text-accent">Sign in</button>
        </p>
      </AuthShell>
    );
  }

  // ── login ──
  return (
    <AuthShell>
      <form onSubmit={submitLogin}>
        <h1 className="font-display font-extrabold leading-tight text-[28px]">Welcome back</h1>
        <p className="text-[14px] text-muted font-semibold mt-1.5 mb-6">Sign in to manage your offers and redemptions.</p>
        <div className="space-y-3.5">
          <Field label="Email" type="email" value={email} onChange={setEmail} icon={Mail} placeholder="you@yourbusiness.lk" autoFocus />
          <Field label="Password" type={showPassword ? 'text' : 'password'} value={password} onChange={setPassword} icon={Lock}
            reveal={showPassword} onReveal={() => setShowPassword((s) => !s)} placeholder="••••••••" />
          <div className="flex items-center justify-between pt-0.5">
            <button type="button" onClick={() => setRemember((r) => !r)} className="flex items-center gap-2">
              <span className={`rounded-md w-5 h-5 flex items-center justify-center border-[1.5px] ${remember ? 'bg-accent border-accent' : 'bg-white border-line'}`}>
                {remember && <Check size={13} strokeWidth={3.5} className="text-white" />}
              </span>
              <span className="text-[13px] font-bold text-ink-soft whitespace-nowrap">Remember me</span>
            </button>
            <button type="button" onClick={() => { setForgotSent(false); setMode('forgot'); }} className="text-[13px] font-extrabold text-accent">Forgot?</button>
          </div>
          {error && <p className="text-[13.5px] font-bold text-coral">{error}</p>}
          <div className="pt-1">
            <Primary type="submit" disabled={busy}>
              {busy ? 'Please wait…' : 'Sign in'}
            </Primary>
          </div>
          <div className="flex items-center gap-3 my-1">
            <div className="flex-1 h-px bg-line" />
            <span className="text-[12px] font-bold text-muted">or</span>
            <div className="flex-1 h-px bg-line" />
          </div>
          <a href={`${API_URL}/auth/google/login`}
            className="w-full flex items-center justify-center gap-2 rounded-2xl py-3 font-extrabold text-[14px] bg-white border-[1.5px] border-line text-ink transition-transform active:scale-[0.97]">
            <GoogleG /> Continue with Google
          </a>
        </div>
        <div className="mt-6 rounded-2xl bg-accent-tint p-4 flex items-center gap-3">
          <Store size={20} className="text-accent shrink-0" />
          <div className="flex-1">
            <p className="text-[13.5px] font-extrabold">New to Spotto?</p>
            <button type="button" onClick={() => { setRegisterStep(0); setMode('register'); setError(null); }} className="text-[13px] font-extrabold text-accent whitespace-nowrap">
              List your business →
            </button>
          </div>
        </div>
      </form>
    </AuthShell>
  );
}

function AuthShell({ children, wide }: { children: React.ReactNode; wide?: boolean }) {
  const features = [
    { icon: Users, t: 'Reach travelers near you', s: 'Get discovered by nomads, surfers & explorers in the bay.' },
    { icon: Tag, t: 'Run offers in seconds', s: 'Launch perks, discounts and bundles — live instantly.' },
    { icon: QrCode, t: 'Validate with a scan', s: 'Redeem codes at the counter, track every claim.' },
  ];
  return (
    <div className="min-h-screen flex bg-sand">
      <div className="hidden md:flex flex-col justify-between relative overflow-hidden shrink-0 p-11"
        style={{ flex: '0 0 46%', background: 'linear-gradient(160deg, var(--accent) 0%, var(--accent-2) 100%)' }}>
        <span className="absolute rounded-full bg-white/10" style={{ right: -60, top: -40, width: 280, height: 280 }} />
        <span className="absolute rounded-full bg-white/[0.08]" style={{ left: -50, bottom: -60, width: 220, height: 220 }} />
        <div className="relative flex items-center gap-2.5">
          <SpottoMark size={30} light />
          <div className="leading-none">
            <p className="font-display font-extrabold text-[20px] tracking-tight text-white">Spotto</p>
            <p className="text-[10px] font-extrabold uppercase tracking-[0.2em] mt-1 text-white/85">Vendor Portal</p>
          </div>
        </div>
        <div className="relative">
          <h2 className="font-display font-extrabold text-white leading-[1.1] text-[32px]">Turn passers-by<br />into regulars.</h2>
          <div className="mt-7 space-y-4">
            {features.map((f) => (
              <div key={f.t} className="flex items-start gap-3.5">
                <span className="w-10 h-10 rounded-xl bg-white/[0.18] text-white flex items-center justify-center shrink-0"><f.icon size={19} /></span>
                <div>
                  <p className="font-extrabold text-[15px] text-white">{f.t}</p>
                  <p className="text-[13px] font-semibold leading-snug text-white/80">{f.s}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
        <div className="relative rounded-2xl px-4 py-3.5 flex items-center gap-3 bg-white/15">
          <TrendingUp size={20} className="text-white shrink-0" />
          <p className="text-[13px] font-bold text-white leading-snug">Partners see <span className="font-extrabold">+18% redemptions</span> in their first month.</p>
        </div>
      </div>

      <div className="flex-1 flex items-center justify-center px-7 py-10 overflow-y-auto">
        <div className="w-full" style={{ maxWidth: wide ? 400 : 380 }}>{children}</div>
      </div>
    </div>
  );
}
