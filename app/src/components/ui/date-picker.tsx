import * as React from "react";
import { format } from "date-fns";

import { cn } from "@/components/ui/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  CalendarIcon,
  ChevronLeft,
  ChevronRight,
  Eraser,
  Minus,
  Plus,
} from "lucide-react";
import { useTranslation } from "react-i18next";

type DatePickerProps = {
  classNameTrigger?: string;
  classNameContent?: string;
  value?: string;
  onValueChange?: (value: string) => void;
};

type CalendarCell = {
  date: Date;
  inMonth: boolean;
  key: string;
};

function parseDate(value?: string): Date | undefined {
  try {
    if (!value) return undefined;
    if (value.includes(" ")) value = value.split(" ")[0]; // Remove time
    const [year, month, day] = value.split("-").map(Number);
    if (!year || !month || !day) return undefined;
    return new Date(year, month - 1, day);
  } catch (e) {
    console.warn("Invalid date format", value, e);
    return undefined;
  }
}

function isSameDay(a?: Date, b?: Date): boolean {
  return (
    !!a &&
    !!b &&
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

function addMonth(date: Date, amount: number): Date {
  return new Date(date.getFullYear(), date.getMonth() + amount, 1);
}

function buildCalendarCells(month: Date): CalendarCell[] {
  const year = month.getFullYear();
  const monthIndex = month.getMonth();
  const firstDay = new Date(year, monthIndex, 1);
  const daysInMonth = new Date(year, monthIndex + 1, 0).getDate();
  const leadingDays = firstDay.getDay();
  const cellCount = Math.ceil((leadingDays + daysInMonth) / 7) * 7;

  return Array.from({ length: cellCount }, (_, index) => {
    const dayOffset = index - leadingDays + 1;
    const date = new Date(year, monthIndex, dayOffset);
    return {
      date,
      inMonth: date.getMonth() === monthIndex,
      key: `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`,
    };
  });
}

const DatePicker = ({
  value,
  onValueChange,
  classNameTrigger,
  classNameContent,
}: DatePickerProps) => {
  const { t } = useTranslation();
  const [open, setOpen] = React.useState(false);
  const date = React.useMemo(() => parseDate(value), [value]);
  const [visibleMonth, setVisibleMonth] = React.useState<Date>(
    () => date ?? new Date(),
  );
  const today = React.useMemo(() => new Date(), []);
  const cells = React.useMemo(
    () => buildCalendarCells(visibleMonth),
    [visibleMonth],
  );
  const weekdays = React.useMemo(() => {
    const formatter = new Intl.DateTimeFormat(undefined, {
      weekday: "short",
    });
    return Array.from({ length: 7 }, (_, index) =>
      formatter.format(new Date(2026, 5, index)),
    );
  }, []);

  React.useEffect(() => {
    if (open) {
      setVisibleMonth(date ?? new Date());
    }
  }, [date, open]);

  const updateDate = React.useCallback(
    (next?: Date) => {
      onValueChange?.(next ? format(next, "yyyy-MM-dd") : "");
      if (next) setOpen(false);
    },
    [onValueChange],
  );

  const addYear = () => {
    const current = date || visibleMonth || new Date();
    updateDate(
      new Date(
        current.getFullYear() + 1,
        current.getMonth(),
        current.getDate(),
      ),
    );
  };

  const subYear = () => {
    const current = date || visibleMonth || new Date();
    updateDate(
      new Date(
        current.getFullYear() - 1,
        current.getMonth(),
        current.getDate(),
      ),
    );
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          unClickable
          variant={"outline"}
          className={cn(
            "h-12 w-[240px] justify-start text-left font-normal",
            !date && "text-muted-foreground",
            classNameTrigger,
          )}
        >
          <CalendarIcon className="mr-2 h-4 w-4" />
          {date ? (
            `${format(date, "yyyy/MM/dd")}`
          ) : (
            <span>{t("date.pick")}</span>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent
        className={cn(
          "w-[300px] max-w-[calc(100vw-1rem)] overflow-hidden rounded-lg border bg-popover p-0 shadow-xl",
          classNameContent,
        )}
        align="start"
        sideOffset={8}
      >
        <div className="px-3 pt-4">
          <div className="relative flex h-9 items-center justify-center">
            <Button
              unClickable
              variant="outline"
              size="icon"
              aria-label="Go to the Previous Month"
              className="absolute left-0 top-0 h-9 w-9 rounded-md bg-background p-0 text-muted-foreground shadow-sm hover:bg-muted hover:text-foreground"
              onClick={() =>
                setVisibleMonth((current) => addMonth(current, -1))
              }
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <div className="text-center text-xl font-semibold leading-none">
              {visibleMonth.getFullYear()}/{visibleMonth.getMonth() + 1}
            </div>
            <Button
              unClickable
              variant="outline"
              size="icon"
              aria-label="Go to the Next Month"
              className="absolute right-0 top-0 h-9 w-9 rounded-md bg-background p-0 text-muted-foreground shadow-sm hover:bg-muted hover:text-foreground"
              onClick={() => setVisibleMonth((current) => addMonth(current, 1))}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
          <div className="sr-only" aria-live="polite">
            {visibleMonth.getFullYear()}/{visibleMonth.getMonth() + 1}
          </div>
          <div className="mt-5 grid grid-cols-7 gap-y-2 gap-x-1">
            {weekdays.map((weekday) => (
              <div
                key={weekday}
                className="flex h-8 w-8 items-center justify-center text-sm font-medium text-muted-foreground tabular-nums"
              >
                {weekday}
              </div>
            ))}
            {cells.map((cell) => {
              const selected = isSameDay(cell.date, date);
              const currentDay = isSameDay(cell.date, today);
              return (
                <button
                  key={cell.key}
                  type="button"
                  aria-pressed={selected}
                  className={cn(
                    "flex h-8 w-8 items-center justify-center justify-self-center rounded-md text-sm font-normal tabular-nums outline-none transition-colors hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
                    !cell.inMonth &&
                      "text-muted-foreground/45 hover:bg-muted hover:text-muted-foreground",
                    currentDay && "bg-muted text-foreground",
                    selected &&
                      "bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground",
                  )}
                  onClick={() => updateDate(cell.date)}
                >
                  {cell.date.getDate()}
                </button>
              );
            })}
          </div>
        </div>
        <div className="flex items-center gap-3 px-3 pb-4 pt-3">
          <Button
            unClickable
            variant="ghost"
            size="icon"
            aria-label="Add one year"
            className="h-8 w-8 rounded-md"
            onClick={addYear}
          >
            <Plus className="h-4 w-4" />
          </Button>
          <Button
            unClickable
            variant="ghost"
            size="icon"
            aria-label="Subtract one year"
            className="h-8 w-8 rounded-md"
            onClick={subYear}
          >
            <Minus className="h-4 w-4" />
          </Button>
          <Button
            unClickable
            variant="outline"
            size="icon"
            aria-label="Clear date"
            className="h-8 w-10 rounded-md"
            onClick={() => updateDate(undefined)}
          >
            <Eraser className="h-4 w-4" />
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
};

export default DatePicker;
