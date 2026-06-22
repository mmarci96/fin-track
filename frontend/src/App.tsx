import { Routes, Route } from 'react-router-dom';
import { AppShell } from '@/components/AppShell';
import { RequireAuth } from '@/auth/RequireAuth';
import { Login } from '@/pages/Login';
import { ReceiptList } from '@/pages/ReceiptList';
import { Scan } from '@/pages/Scan';
import { ReceiptDetail } from '@/pages/ReceiptDetail';
import { Settings } from '@/pages/Settings';

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/*"
        element={
          <RequireAuth>
            <AppShell>
              <Routes>
                <Route path="/" element={<ReceiptList />} />
                <Route path="/scan" element={<Scan />} />
                <Route path="/receipts/:id" element={<ReceiptDetail />} />
                <Route path="/settings" element={<Settings />} />
              </Routes>
            </AppShell>
          </RequireAuth>
        }
      />
    </Routes>
  );
}
