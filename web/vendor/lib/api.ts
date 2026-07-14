'use client';

import { createApi } from '@spotto/shared';

/** Vendor portal API client — token in localStorage under its own key. */
export const api = createApi('spotto_vendor_token');
