# UI Solution: UI-SOL-PLAT-003 — SSO Google OAuth UI

**Solution ID:** UI-SOL-PLAT-003  
**CR References:** [CR-PLAT-003](../../../../docs/crs/v3/platform/CR-PLAT-003-SSO-Google-OAuth.md)  
**Feature:** SSO — Google OAuth Login Button + Callback Handling  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/login/`, `ui/src/pages/login/oauth-callback/`

---

## 1. Mục Đích

Implement Google SSO UI:
- Google login button trên login page
- OAuth redirect flow (popup hoặc redirect)
- Callback page xử lý authorization code
- Error handling cho SSO failures

---

## 2. Backend API Contract

```http
# Initiate Google OAuth flow
GET /v1/auth/google/authorize
→ Redirect to Google OAuth consent screen

# Callback (after Google redirects back)
GET /v1/auth/google/callback?code=...&state=...
→ { access_token, refresh_token, expires_in, token_type, user }
  (same as LoginResponse)

# Token-based (for popup flow)
POST /v1/auth/google/token
{ "code": string, "redirect_uri": string }
→ LoginResponse
```

---

## 3. Components

### 3.1 Login Page Enhancement

```typescript
// ui/src/pages/login/LoginPage.tsx

export function LoginPage() {
  return (
    <div className="login-form">
      {/* Existing email/password form */}
      <EmailPasswordForm />
      
      <Divider text="or" />
      
      {/* Google SSO Button */}
      <GoogleLoginButton
        onClick={handleGoogleLogin}
        disabled={!isGoogleSSOEnabled}   // feature flag from config
      />
      
      {!isGoogleSSOEnabled && (
        <p className="text-xs text-gray-400 text-center">
          Google login coming soon
        </p>
      )}
    </div>
  );
}
```

### 3.2 Google Login Button

```typescript
// Styled Google button with official Google branding guidelines
function GoogleLoginButton({ onClick, disabled }) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className="flex items-center gap-3 w-full border border-gray-300 
                 rounded-lg px-4 py-2.5 hover:bg-gray-50 transition-colors"
    >
      <GoogleIcon className="w-5 h-5" />
      <span>Continue with Google</span>
    </button>
  );
}
```

### 3.3 OAuth Flow (Redirect Method)

```typescript
// 1. Click → redirect to backend authorize URL
function handleGoogleLogin() {
  const returnUrl = encodeURIComponent(window.location.pathname);
  window.location.href = `/v1/auth/google/authorize?return_to=${returnUrl}`;
}

// 2. OAuth Callback Page (/login/callback)
// ui/src/pages/login/OAuthCallbackPage.tsx
export function OAuthCallbackPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  
  useEffect(() => {
    const code  = searchParams.get('code');
    const error = searchParams.get('error');
    
    if (error) {
      navigate('/login?error=oauth_failed');
      return;
    }
    
    if (code) {
      // Exchange code via backend callback endpoint (handled server-side redirect)
      // Token already set in response → just read from localStorage
      const token = localStorage.getItem('access_token');
      if (token) navigate('/dashboard');
      else navigate('/login?error=token_missing');
    }
  }, [searchParams, navigate]);
  
  return <LoadingSpinner text="Signing in with Google..." />;
}
```

---

## 4. Feature Flag

```typescript
// ui/src/config/features.ts
export const FEATURES = {
  GOOGLE_SSO_ENABLED: import.meta.env.VITE_GOOGLE_SSO_ENABLED === 'true',
};
// When false: Google button shown as disabled with "coming soon" tooltip
```

---

## 5. Acceptance Criteria (Frontend)

- [ ] Google button visible on login page
- [ ] When SSO disabled: button grayed out + "coming soon" tooltip
- [ ] When SSO enabled: click redirects to Google consent screen
- [ ] Callback page: handles `code` param → stores tokens → redirects
- [ ] Error handling: `?error=access_denied` → toast "Google login cancelled"
- [ ] Loading spinner shown during OAuth callback processing
