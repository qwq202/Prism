import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Filter, Loader2 } from "lucide-react";
import { UserTypeChartResponse } from "@/admin/types.ts";
import Tips from "@/components/Tips.tsx";
import { ChartLegend, DonutChart } from "./recharts.tsx";
import { Button } from "@/components/ui/button.tsx";
import { MultiCombobox } from "@/components/ui/multi-combobox.tsx";
import { getReadableNumber } from "@/utils/processor.ts";

type UserTypeChartProps = {
  data: UserTypeChartResponse;
};

enum UserType {
  normal = "normal",
  api_paid = "api_paid",
  basic_plan = "basic_plan",
  standard_plan = "standard_plan",
  pro_plan = "pro_plan",
}

type UserStatus = {
  name: string;
  value: number;
};

function UserTypeChart({ data }: UserTypeChartProps) {
  const { t } = useTranslation();
  const identityLabels = useMemo(
    () => ({
      [UserType.normal]: t("admin.identity.normal"),
      [UserType.api_paid]: t("admin.identity.api_paid"),
      [UserType.basic_plan]: t("admin.identity.basic_plan"),
      [UserType.standard_plan]: t("admin.identity.standard_plan"),
      [UserType.pro_plan]: t("admin.identity.pro_plan"),
    }),
    [t],
  );

  const [display, setDisplay] = useState<UserType[]>([
    UserType.normal,
    UserType.api_paid,
    UserType.basic_plan,
    UserType.standard_plan,
    UserType.pro_plan,
  ]);

  const chart = useMemo((): UserStatus[] => {
    return [
      display.includes(UserType.normal) && {
        name: identityLabels[UserType.normal],
        value: data.normal,
      },
      display.includes(UserType.api_paid) && {
        name: identityLabels[UserType.api_paid],
        value: data.api_paid,
      },
      display.includes(UserType.basic_plan) && {
        name: identityLabels[UserType.basic_plan],
        value: data.basic_plan,
      },
      display.includes(UserType.standard_plan) && {
        name: identityLabels[UserType.standard_plan],
        value: data.standard_plan,
      },
      display.includes(UserType.pro_plan) && {
        name: identityLabels[UserType.pro_plan],
        value: data.pro_plan,
      },
    ].filter((item) => item) as UserStatus[];
  }, [display, data, identityLabels]);

  const total = useMemo(
    () => chart.reduce((sum, item) => sum + item.value, 0),
    [chart],
  );

  const colors = ["blue", "cyan", "indigo", "violet", "fuchsia"];

  return (
    <div className={`chart`}>
      <div className={`chart-title mb-2`}>
        <div className={`flex flex-row items-center w-full`}>
          <div>
            {t("admin.user-type-chart")}
            <Tips
              className={`translate-y-[2px]`}
              content={t("admin.user-type-chart-tip")}
            />
          </div>
          {data.total === 0 && (
            <Loader2 className={`h-4 w-4 ml-1 animate-spin`} />
          )}

          <div className={`grow`} />
          <MultiCombobox
            value={display}
            align={`end`}
            onChange={(value) => setDisplay(value as UserType[])}
            list={[
              UserType.normal,
              UserType.api_paid,
              UserType.basic_plan,
              UserType.standard_plan,
              UserType.pro_plan,
            ]}
            listTranslate={`admin.identity`}
          >
            <Button variant={`ghost`} size={`icon-sm`}>
              <Filter className={`h-4 w-4`} />
            </Button>
          </MultiCombobox>
        </div>
      </div>
      <div className={`flex flex-row`}>
        <DonutChart
          className={`common-chart p-2 w-[50%]`}
          data={chart}
          colors={colors}
          centerLabel={getReadableNumber(total)}
        />
        <ChartLegend
          className={`common-chart p-4 w-[50%]`}
          categories={chart.map((item) => item.name)}
          colors={colors}
        />
      </div>
    </div>
  );
}

export default UserTypeChart;
