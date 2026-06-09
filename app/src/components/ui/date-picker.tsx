import * as React from "react";
import { format } from "date-fns";

import { cn } from "@/components/ui/lib/utils";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { CalendarIcon, Eraser, Minus, Plus } from "lucide-react";
import { useTranslation } from "react-i18next";

type DatePickerProps = Omit<
  React.ComponentProps<typeof Calendar>,
  "mode" | "selected" | "onSelect" | "required"
> & {
  classNameTrigger?: string;
  classNameContent?: string;
  value?: string;
  onValueChange?: (value: string) => void;
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

const DatePicker = ({
  value,
  onValueChange,
  classNameTrigger,
  classNameContent,
  ...props
}: DatePickerProps) => {
  const { t } = useTranslation();
  const [open, setOpen] = React.useState(false);
  const date = React.useMemo(() => parseDate(value), [value]);

  const updateDate = React.useCallback(
    (next?: Date) => {
      onValueChange?.(next ? format(next, "yyyy-MM-dd") : "");
      if (next) setOpen(false);
    },
    [onValueChange],
  );

  const addYear = () => {
    const current = date || new Date();
    updateDate(
      new Date(
        current.getFullYear() + 1,
        current.getMonth(),
        current.getDate(),
      ),
    );
  };

  const subYear = () => {
    const current = date || new Date();
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
          "w-[292px] max-w-[calc(100vw-2rem)] rounded-lg p-0 shadow-lg",
          classNameContent,
        )}
        align="start"
        sideOffset={8}
      >
        <Calendar
          mode="single"
          selected={date}
          onSelect={(date) => updateDate(date)}
          autoFocus
          className="p-3 pb-1"
          classNames={{
            months: "flex w-full flex-col space-y-0",
            month: "w-full space-y-3",
            month_caption: "relative flex h-8 items-center justify-center",
            caption_label: "text-base font-semibold leading-none",
            nav: "absolute inset-x-0 top-0 flex h-8 items-center justify-between",
            button_previous:
              "absolute left-0 top-0 flex h-8 w-8 items-center justify-center rounded-md border bg-background p-0 text-muted-foreground opacity-100 shadow-sm transition-colors hover:bg-muted hover:text-foreground",
            button_next:
              "absolute right-0 top-0 flex h-8 w-8 items-center justify-center rounded-md border bg-background p-0 text-muted-foreground opacity-100 shadow-sm transition-colors hover:bg-muted hover:text-foreground",
            month_grid: "w-full border-collapse table-fixed",
            weekdays: "grid grid-cols-7 gap-1",
            weekday:
              "flex h-7 items-center justify-center text-sm font-normal text-muted-foreground tabular-nums",
            week: "grid grid-cols-7 gap-1",
            day: "flex h-8 w-8 items-center justify-center p-0 text-center",
            day_button:
              "h-8 w-8 rounded-md p-0 text-sm font-normal transition-colors hover:bg-muted",
            selected:
              "bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground focus:bg-primary focus:text-primary-foreground",
            today: "bg-muted text-foreground",
            outside:
              "text-muted-foreground/50 opacity-100 aria-selected:bg-primary aria-selected:text-primary-foreground",
            disabled: "text-muted-foreground/40 opacity-100",
            hidden: "invisible",
          }}
          {...props}
        />
        <div className="flex items-center gap-3 px-3 pb-3 pt-1">
          <Button
            unClickable
            variant="ghost"
            size="icon"
            className="h-8 w-8 rounded-md"
            onClick={addYear}
          >
            <Plus className="h-4 w-4" />
          </Button>
          <Button
            unClickable
            variant="ghost"
            size="icon"
            className="h-8 w-8 rounded-md"
            onClick={subYear}
          >
            <Minus className="h-4 w-4" />
          </Button>
          <Button
            unClickable
            variant="outline"
            size="icon"
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
