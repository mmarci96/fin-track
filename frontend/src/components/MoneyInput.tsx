import { forwardRef } from 'react';
import { Input, type InputProps } from '@/components/ui/input';

// A numeric text input tuned for entering money on mobile (decimal keypad).
// It deals in display strings; the parent converts to/from integer minor units
// via lib/format. Kept dumb so it composes inside react-hook-form.
export type MoneyInputProps = Omit<InputProps, 'type' | 'inputMode'>;

export const MoneyInput = forwardRef<HTMLInputElement, MoneyInputProps>(
  (props, ref) => (
    <Input
      ref={ref}
      type="text"
      inputMode="numeric"
      placeholder="0"
      {...props}
    />
  ),
);
MoneyInput.displayName = 'MoneyInput';
