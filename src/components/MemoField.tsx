import React from 'react';
import { validateMemo } from '../../packages/core-ts/src/routing/memo';
import { KnownMemoType } from '../../packages/core-ts/src/routing/types';

interface MemoFieldProps {
  isVisible: boolean;
  value: string;
  onChange: (value: string) => void;
  memoType?: KnownMemoType;
}

export const MemoField: React.FC<MemoFieldProps> = ({
  isVisible,
  value,
  onChange,
  memoType = 'text',
}) => {
  const uid = React.useId();
  const inputId = `${uid}-memo-input`;
  const counterId = `${uid}-memo-counter`;
  const errorId = `${uid}-memo-error`;

  if (!isVisible) return null;

  const validation = validateMemo(value, memoType);
  const errorMessage = validation.error ? validation.error.message : null;
  const showCounter = validation.maxLength !== null;
  const counterValue =
    validation.lengthUnit === 'bytes' ? validation.byteLength : value.length;
  const describedBy = [
    errorMessage ? errorId : null,
    showCounter ? counterId : null,
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <div style={{ marginTop: '0.75rem', display: 'flex', flexDirection: 'column' }}>
      <label 
        htmlFor={inputId} 
        style={{ fontSize: '0.875rem', fontWeight: '500', color: '#475569', marginBottom: '0.25rem' }}
      >
        Memo (Optional)
      </label>
      <input
        id={inputId}
        type="text"
        placeholder="Enter memo here"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        aria-invalid={!validation.valid}
        aria-describedby={describedBy || undefined}
        style={{
          width: '100%',
          padding: '0.5rem 0.75rem',
          border: `1px solid ${validation.valid ? '#cbd5e1' : '#dc2626'}`,
          borderRadius: '0.375rem',
          fontSize: '1rem',
          boxSizing: 'border-box',
          outline: 'none',
        }}
      />
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'baseline',
          gap: '0.5rem',
          marginTop: '0.25rem',
        }}
      >
        <div style={{ fontSize: '0.75rem', color: '#64748b' }}>
          A memo is sometimes required by exchanges.
        </div>
        {showCounter && (
          <div
            id={counterId}
            aria-live="polite"
            style={{
              fontSize: '0.75rem',
              fontVariantNumeric: 'tabular-nums',
              whiteSpace: 'nowrap',
              color: validation.valid ? '#64748b' : '#dc2626',
            }}
          >
            {counterValue} / {validation.maxLength} {validation.lengthUnit}
          </div>
        )}
      </div>
      {errorMessage && (
        <div
          id={errorId}
          role="status"
          style={{ fontSize: '0.75rem', color: '#dc2626', marginTop: '0.25rem' }}
        >
          {errorMessage}
        </div>
      )}
    </div>
  );
};
