import * as React from "react";
import { Input, InputProps } from "@/components/ui/input.tsx";
import { getNumber } from "@/utils/base.ts";
import { useCallback, useEffect, useMemo, useState } from "react";
import { cn } from "@/components/ui/lib/utils.ts";

export interface NumberInputProps extends InputProps {
  value: number;
  max?: number;
  min?: number;
  onValueChange: (value: number) => void;
  acceptNegative?: boolean;
  acceptNaN?: boolean;
}

const NumberInput = React.forwardRef<HTMLInputElement, NumberInputProps>(
  (
    {
      className,
      onValueChange,
      acceptNaN,
      acceptNegative,
      min,
      max,
      value: propValue,
      ...inputProps
    },
    ref,
  ) => {
    const [value, setValue] = useState(propValue.toString());

    const getValue = useCallback((v: string) => {
      const raw = getNumber(v, acceptNegative);
      let val = parseFloat(raw);
      if (isNaN(val) && !acceptNaN) val = 0;
      if (max !== undefined && val > max) val = max;
      else if (min !== undefined && val < min) val = min;
      return val;
    }, [acceptNaN, acceptNegative, max, min]);

    useEffect(() => {
      // fix life cycle: update value when controlled value changed
      setValue((current) =>
        getValue(current.toString()) !== propValue
          ? propValue.toString()
          : current,
      );
    }, [getValue, propValue]);

    const formatValue = (v: string) => {
      if (v.trim().length === 0) return v.trim();

      if (!/^[-+]?(?:[0-9]*(?:\.[0-9]*)?)?$/.test(v)) {
        const exp = /[-+]?[0-9]+(\.[0-9]+)?/g;
        return v.match(exp)?.join("") || "";
      }

      if (v === "-" && acceptNegative) return v;

      // replace -0124.5 to -124.5, 0043 to 43, 2.000 to 2.000
      const exp = /^[-+]?0+(?=[0-9]+(\.[0-9]+)?$)/;
      v = v.replace(exp, "");

      const raw = getNumber(v, acceptNegative);
      const val = parseFloat(raw);
      if (isNaN(val) && !acceptNaN) return (min ?? 0).toString();
      if (max !== undefined && val > max) return max.toString();
      else if (min !== undefined && val < min) return min.toString();

      return v;
    };

    const isValid = useMemo((): boolean => {
      if (!/^[-+]?[0-9]+(\.[0-9]+)?$/.test(value)) return false;
      const val = getValue(value);
      if (max !== undefined && val > max) return false;
      else if (min !== undefined && val < min) return false;
      return true;
    }, [getValue, max, min, value]);

    return (
      <Input
        {...inputProps}
        ref={ref}
        className={cn(
          "number-input transition",
          className,
          !isValid && "border-red-600 focus:border-red-700",
        )}
        value={value}
        onChange={(e) => {
          setValue(formatValue(e.target.value));
          onValueChange(getValue(e.target.value));
        }}
        min={min}
        max={max}
        onWheel={(e) => {
          e.stopPropagation();
        }}
      />
    );
  },
);
NumberInput.displayName = "NumberInput";

export { NumberInput };
