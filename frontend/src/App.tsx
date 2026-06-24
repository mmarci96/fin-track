import { Routes, Route } from 'react-router-dom';
import { AppShell } from '@/components/AppShell';
import { RequireAuth } from '@/auth/RequireAuth';
import { Login } from '@/pages/Login';
import { ReceiptList } from '@/pages/ReceiptList';
import { Scan } from '@/pages/Scan';
import { ReceiptDetail } from '@/pages/ReceiptDetail';
import { AddExpense } from '@/pages/AddExpense';
import { Statistics } from '@/pages/Statistics';
import { Settings } from '@/pages/Settings';
import { DebugReceipts } from '@/pages/DebugReceipts';

// import.meta.env.DEV is true under `npm run dev` (make dev) and false in the
// production build `npm run build` (make start). It gates developer-only
// affordances so they never ship in the UAT/prod image.
const IS_DEV = import.meta.env.DEV;

export default function App() {
  return (
    <Routes>
      {/* The manual user-id login is a DEV-ONLY testing affordance. Production
          builds authenticate via the auth-service JWT cookie — which traefikauth
          verifies and turns into the trusted X-User-ID header — so this route is
          not registered there at all. */}
      {IS_DEV && <Route path="/login" element={<Login />} />}
      <Route
        path="/*"
        element={
          <RequireAuth>
            <AppShell>
              <Routes>
                <Route path="/" element={<ReceiptList />} />
                <Route path="/scan" element={<Scan />} />
                <Route path="/expenses/new" element={<AddExpense />} />
                <Route path="/statistics" element={<Statistics />} />
                <Route path="/receipts/:id" element={<ReceiptDetail />} />
                <Route path="/settings" element={<Settings />} />
                <Route path="/debug" element={<DebugReceipts />} />
              </Routes>
            </AppShell>
          </RequireAuth>
        }
      />
    </Routes>
  );
}
