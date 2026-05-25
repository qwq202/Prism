import { useTranslation } from "react-i18next";
import { useMemo, useReducer, useState } from "react";
import {
  CommonResponse,
  UserData,
  UserForm,
  UserResponse,
} from "@/admin/types.ts";
import {
  banUserOperation,
  BatchUserAction,
  batchUserOperation,
  createUserOperation,
  deleteUserOperation,
  getUserList,
  initialUserFilter,
  quotaOperation,
  type ReleaseUsageType,
  releaseUsageOperation,
  setAdminOperation,
  subscriptionLevelOperation,
  subscriptionOperation,
  updateEmail,
  updatePassword,
  UserFilterProps,
} from "@/admin/api/chart.ts";
import { useEffectAsync } from "@/utils/hook.ts";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table.tsx";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  ArrowDownNarrowWide,
  Ban,
  CalendarCheck2,
  CalendarClock,
  CalendarPlus,
  CalendarRange,
  Clock,
  CloudCog,
  CloudFog,
  Filter,
  KeyRound,
  Loader2,
  Mail,
  MinusCircle,
  MoreHorizontal,
  PlusCircle,
  RotateCw,
  Search,
  Shield,
  ShieldMinus,
  Trash2,
} from "lucide-react";
import { Input } from "@/components/ui/input.tsx";
import PopupDialog, { popupTypes } from "@/components/PopupDialog.tsx";
import { getNumber, isEnter, parseNumber } from "@/utils/base.ts";
import { useDispatch, useSelector } from "react-redux";
import { selectUsername, validateToken } from "@/store/auth.ts";
import { PaginationAction } from "@/components/ui/pagination.tsx";
import Tips from "@/components/Tips.tsx";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTrigger,
  DialogTitle,
  DialogFooter,
  DialogAction,
  DialogCancel,
} from "@/components/ui/dialog.tsx";
import { RadioBox } from "@/components/ui/radio-box.tsx";
import { formReducer, isEmailValid } from "@/utils/form.ts";
import { Separator } from "@/components/ui/separator.tsx";
import { toast } from "sonner";
import { Badge } from "../ui/badge";
import type { TFunction } from "i18next";
import { Checkbox } from "@/components/ui/checkbox.tsx";

type OperationMenuProps = {
  user: UserData;
  onRefresh?: () => void;
};

type CreateUserFormState = {
  username: string;
  email: string;
  password: string;
};

enum UserType {
  normal = "normal",
  basic_plan = "basic_plan",
  standard_plan = "standard_plan",
  pro_plan = "pro_plan",
}

const userTypeArray = [
  UserType.normal,
  UserType.basic_plan,
  UserType.standard_plan,
  UserType.pro_plan,
];

function doToast(t: TFunction, resp: CommonResponse) {
  if (!resp.status)
    toast.error(t("admin.operate-failed"), {
      description: t("admin.operate-failed-prompt", {
        reason: resp.message || resp.error,
      }),
    });
  else
    toast.success(t("admin.operate-success"), {
      description: t("admin.operate-success-prompt"),
    });
}

function OperationMenu({ user, onRefresh }: OperationMenuProps) {
  const { t } = useTranslation();
  const dispatch = useDispatch();

  const username = useSelector(selectUsername);

  const [passwordOpen, setPasswordOpen] = useState<boolean>(false);
  const [emailOpen, setEmailOpen] = useState<boolean>(false);
  const [quotaOpen, setQuotaOpen] = useState<boolean>(false);
  const [quotaSetOpen, setQuotaSetOpen] = useState<boolean>(false);
  const [subscriptionOpen, setSubscriptionOpen] = useState<boolean>(false);
  const [subscriptionLevelOpen, setSubscriptionLevelOpen] =
    useState<boolean>(false);
  const [releaseHourOpen, setReleaseHourOpen] = useState<boolean>(false);
  const [releaseWeekOpen, setReleaseWeekOpen] = useState<boolean>(false);
  const [banOpen, setBanOpen] = useState<boolean>(false);
  const [adminOpen, setAdminOpen] = useState<boolean>(false);
  const [deleteOpen, setDeleteOpen] = useState<boolean>(false);

  const handleReleaseUsage = async (type: ReleaseUsageType) => {
    const resp = await releaseUsageOperation(user.id, type);
    doToast(t, resp);

    if (resp.status) onRefresh?.();
    return resp.status;
  };

  return (
    <>
      <PopupDialog
        destructive={true}
        type={popupTypes.Password}
        title={t("admin.password-action")}
        name={t("auth.password")}
        description={t("admin.password-action-desc")}
        open={passwordOpen}
        setOpen={setPasswordOpen}
        defaultValue={""}
        onSubmit={async (password) => {
          const resp = await updatePassword(user.id, password);
          doToast(t, resp);

          if (resp.status) {
            if (username === user.username) {
              if (resp.token) validateToken(dispatch, resp.token);
              else location.reload();
            }
            onRefresh?.();
          }

          return resp.status;
        }}
      />
      <PopupDialog
        destructive={true}
        type={popupTypes.Text}
        title={t("admin.email-action")}
        name={t("admin.email")}
        description={t("admin.email-action-desc")}
        open={emailOpen}
        setOpen={setEmailOpen}
        defaultValue={user.email}
        onSubmit={async (email) => {
          const resp = await updateEmail(user.id, email);
          doToast(t, resp);

          if (resp.status) onRefresh?.();
          return resp.status;
        }}
      />
      <PopupDialog
        type={popupTypes.Number}
        title={t("admin.quota-action")}
        name={t("admin.quota")}
        description={t("admin.quota-action-desc")}
        defaultValue={"0"}
        onValueChange={getNumber}
        open={quotaOpen}
        setOpen={setQuotaOpen}
        componentProps={{ acceptNegative: true }}
        onSubmit={async (value) => {
          const quota = parseNumber(value);
          const resp = await quotaOperation(user.id, quota);
          doToast(t, resp);

          if (resp.status) onRefresh?.();
          return resp.status;
        }}
      />
      <PopupDialog
        type={popupTypes.Number}
        title={t("admin.quota-set-action")}
        name={t("admin.quota")}
        description={t("admin.quota-set-action-desc")}
        defaultValue={user.quota.toFixed(2)}
        onValueChange={getNumber}
        open={quotaSetOpen}
        setOpen={setQuotaSetOpen}
        componentProps={{ acceptNegative: true }}
        onSubmit={async (value) => {
          const quota = parseNumber(value);
          const resp = await quotaOperation(user.id, quota, true);
          doToast(t, resp);

          if (resp.status) onRefresh?.();
          return resp.status;
        }}
      />
      <PopupDialog
        type={popupTypes.Clock}
        title={t("admin.subscription-action")}
        description={t("admin.subscription-action-desc", {
          username: user.username,
        })}
        defaultValue={
          user.expired_at ||
          new Date().toISOString().slice(0, 19).replace("T", " ")
        }
        open={subscriptionOpen}
        setOpen={setSubscriptionOpen}
        onSubmit={async (value) => {
          const resp = await subscriptionOperation(user.id, value);
          doToast(t, resp);

          if (resp.status) onRefresh?.();
          return resp.status;
        }}
      />
      <PopupDialog
        type={popupTypes.List}
        title={t("admin.subscription-level")}
        name={t("admin.level")}
        description={t("admin.subscription-level-desc")}
        defaultValue={userTypeArray[user.level]}
        params={{
          dataList: userTypeArray,
          dataListTranslated: "admin.identity",
        }}
        open={subscriptionLevelOpen}
        setOpen={setSubscriptionLevelOpen}
        onSubmit={async (value) => {
          const level = userTypeArray.indexOf(value as UserType);
          const resp = await subscriptionLevelOperation(user.id, level);
          doToast(t, resp);

          if (resp.status) onRefresh?.();
          return resp.status;
        }}
      />
      <PopupDialog
        type={popupTypes.Empty}
        title={t("admin.reset-hour-limit-action")}
        description={t("admin.reset-hour-limit-action-desc")}
        open={releaseHourOpen}
        setOpen={setReleaseHourOpen}
        onSubmit={() => handleReleaseUsage("hour")}
      />
      <PopupDialog
        type={popupTypes.Empty}
        title={t("admin.reset-weekly-limit-action")}
        description={t("admin.reset-weekly-limit-action-desc")}
        open={releaseWeekOpen}
        setOpen={setReleaseWeekOpen}
        onSubmit={() => handleReleaseUsage("week")}
      />
      <PopupDialog
        disabled={username === user.username}
        destructive={true}
        type={popupTypes.Empty}
        title={user.is_banned ? t("admin.unban-action") : t("admin.ban-action")}
        description={
          user.is_banned
            ? t("admin.unban-action-desc")
            : t("admin.ban-action-desc")
        }
        open={banOpen}
        setOpen={setBanOpen}
        onSubmit={async () => {
          const resp = await banUserOperation(user.id, !user.is_banned);
          doToast(t, resp);

          if (resp.status) onRefresh?.();
          return resp.status;
        }}
      />
      <PopupDialog
        disabled={username === user.username}
        destructive={true}
        type={popupTypes.Empty}
        title={
          user.is_admin
            ? t("admin.cancel-admin-action")
            : t("admin.set-admin-action")
        }
        description={
          user.is_admin
            ? t("admin.cancel-admin-action-desc")
            : t("admin.set-admin-action-desc")
        }
        open={adminOpen}
        setOpen={setAdminOpen}
        onSubmit={async () => {
          const resp = await setAdminOperation(user.id, !user.is_admin);
          doToast(t, resp);

          if (resp.status) onRefresh?.();
          return resp.status;
        }}
      />
      <PopupDialog
        disabled={username === user.username}
        destructive={true}
        type={popupTypes.Empty}
        title={t("admin.delete-user-action")}
        description={t("admin.delete-user-action-desc", {
          username: user.username,
        })}
        confirmLabel={t("admin.delete-user-confirm")}
        open={deleteOpen}
        setOpen={setDeleteOpen}
        onSubmit={async () => {
          const resp = await deleteUserOperation(user.id);
          doToast(t, resp);

          if (resp.status) onRefresh?.();
          return resp.status;
        }}
      />

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant={`outline`} size={`icon`}>
            <MoreHorizontal className={`h-4 w-4`} />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent className={`min-w-[8.75rem]`}>
          <DropdownMenuItem onClick={() => setPasswordOpen(true)}>
            <KeyRound className={`h-4 w-4 mr-2`} />
            {t("admin.password-action")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setEmailOpen(true)}>
            <Mail className={`h-4 w-4 mr-2`} />
            {t("admin.email-action")}
          </DropdownMenuItem>
          {user.is_banned ? (
            <DropdownMenuItem onClick={() => setBanOpen(true)}>
              <PlusCircle className={`h-4 w-4 mr-2`} />
              {t("admin.unban-action")}
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem onClick={() => setBanOpen(true)}>
              <MinusCircle className={`h-4 w-4 mr-2`} />
              {t("admin.ban-action")}
            </DropdownMenuItem>
          )}
          {user.is_admin ? (
            <DropdownMenuItem onClick={() => setAdminOpen(true)}>
              <ShieldMinus className={`h-4 w-4 mr-2`} />
              {t("admin.cancel-admin-action")}
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem onClick={() => setAdminOpen(true)}>
              <Shield className={`h-4 w-4 mr-2`} />
              {t("admin.set-admin-action")}
            </DropdownMenuItem>
          )}
          <DropdownMenuItem onClick={() => setQuotaOpen(true)}>
            <CloudFog className={`h-4 w-4 mr-2`} />
            {t("admin.quota-action")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setQuotaSetOpen(true)}>
            <CloudCog className={`h-4 w-4 mr-2`} />
            {t("admin.quota-set-action")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setSubscriptionOpen(true)}>
            <CalendarClock className={`h-4 w-4 mr-2`} />
            {t("admin.subscription-action")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setSubscriptionLevelOpen(true)}>
            <CalendarCheck2 className={`h-4 w-4 mr-2`} />
            {t("admin.subscription-level")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setReleaseHourOpen(true)}>
            <Clock className={`h-4 w-4 mr-2`} />
            {t("admin.reset-hour-limit-action")}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setReleaseWeekOpen(true)}>
            <CalendarRange className={`h-4 w-4 mr-2`} />
            {t("admin.reset-weekly-limit-action")}
          </DropdownMenuItem>
          <DropdownMenuItem
            disabled={username === user.username}
            className="text-destructive focus:text-destructive"
            onClick={() => setDeleteOpen(true)}
          >
            <Trash2 className={`h-4 w-4 mr-2`} />
            {t("admin.delete-user-action")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}

function UserTable() {
  const { t } = useTranslation();
  const [data, setData] = useState<UserForm>({
    total: 0,
    data: [],
  });
  const [page, setPage] = useState<number>(0);
  const [search, setSearch] = useState<string>("");
  const [loading, setLoading] = useState<boolean>(false);
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [batchQuota, setBatchQuota] = useState<string>("5");
  const [createOpen, setCreateOpen] = useState<boolean>(false);
  const [createForm, setCreateForm] = useState<CreateUserFormState>({
    username: "",
    email: "",
    password: "",
  });

  const [filter, filterDispatch] = useReducer(formReducer<UserFilterProps>(), {
    ...initialUserFilter,
  });

  const [filterDialog, setFilterDialog] = useState<boolean>(false);

  const filterConds = useMemo((): number => {
    return Object.values(filter).filter(
      (value) => value !== "all" && value !== "id-asc",
    ).length;
  }, [filter]);

  async function update() {
    setLoading(true);
    const resp = await getUserList(page, search, filter);
    setLoading(false);
    if (resp.status) {
      setData(resp as UserResponse);
      setSelected(new Set());
    } else
      toast.error(t("admin.error"), {
        description: resp.message,
      });
  }

  async function executeBatch(action: BatchUserAction, value?: number) {
    const ids = Array.from(selected);
    const resp = await batchUserOperation(ids, action, value);
    if (resp.status) {
      toast.success(t("admin.batch-success", { count: ids.length }));
      await update();
    } else {
      toast.error(t("admin.error"), { description: resp.message });
    }
  }

  const allChecked =
    data.data.length > 0 && data.data.every((u) => selected.has(u.id));
  const batchActionClass = "h-10 w-24 px-3 text-sm shrink-0";
  const batchQuotaInputClass = "h-10 w-24 text-sm shrink-0";

  const updateCreateForm = (key: keyof CreateUserFormState, value: string) => {
    setCreateForm((prev) => ({ ...prev, [key]: value }));
  };

  const resetCreateForm = () => {
    setCreateForm({ username: "", email: "", password: "" });
  };

  async function createUser() {
    const username = createForm.username.trim();
    const email = createForm.email.trim();
    const password = createForm.password.trim();

    if (!username || !email || !password) {
      toast.error(t("admin.operate-failed"), {
        description: t("admin.add-user-required"),
      });
      return;
    }

    if (username.length < 2 || username.length > 24) {
      toast.error(t("admin.operate-failed"), {
        description: t("admin.add-user-username-invalid"),
      });
      return;
    }

    if (!isEmailValid(email)) {
      toast.error(t("admin.operate-failed"), {
        description: t("auth.invalid-email"),
      });
      return;
    }

    if (password.length < 6 || password.length > 36) {
      toast.error(t("admin.operate-failed"), {
        description: t("admin.add-user-password-invalid"),
      });
      return;
    }

    const resp = await createUserOperation(username, email, password);
    doToast(t, resp);

    if (resp.status) {
      setCreateOpen(false);
      resetCreateForm();
      await update();
    }
  }

  function toggleAll() {
    if (allChecked) {
      setSelected(new Set());
    } else {
      setSelected(new Set(data.data.map((u) => u.id)));
    }
  }

  function toggleOne(id: number) {
    setSelected((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  }
  useEffectAsync(update, [page]);

  return (
    <div className={`user-table`}>
      <div className={`flex flex-row mb-6`}>
        <Dialog open={filterDialog} onOpenChange={setFilterDialog}>
          <DialogTrigger asChild>
            <Button size={`icon`} className={`shrink-0 mr-2`} onClick={update}>
              <Filter className={`h-4 w-4`} />
              {filterConds > 0 && (
                <span className={`text-xs`}>{filterConds}</span>
              )}
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogTitle>{t("filter.filter")}</DialogTitle>
            <DialogDescription>
              {t("filter.conds", { count: filterConds })}
            </DialogDescription>
            <div className={`flex flex-col`}>
              <RadioBox
                title={t("filter.admin")}
                icon={<Shield />}
                prefix="admin"
                items={[
                  { id: "all", value: t("filter.all") },
                  { id: "yes", value: t("filter.admin") },
                  { id: "no", value: t("filter.not-admin") },
                ]}
                value={filter.admin}
                onValueChange={(value) =>
                  filterDispatch({ type: "update:admin", value })
                }
              />
              <RadioBox
                title={t("filter.ban")}
                icon={<ShieldMinus />}
                prefix="ban"
                items={[
                  { id: "all", value: t("filter.all") },
                  { id: "yes", value: t("filter.banned") },
                  { id: "no", value: t("filter.not-banned") },
                ]}
                value={filter.ban}
                onValueChange={(value) =>
                  filterDispatch({ type: "update:ban", value })
                }
              />
              <RadioBox
                title={t("filter.plan")}
                icon={<CalendarPlus />}
                prefix="plan"
                items={[
                  { id: "all", value: t("filter.all") },
                  { id: "yes", value: t("filter.subscribed") },
                  { id: "no", value: t("filter.unsubscribed") },
                ]}
                value={filter.plan}
                onValueChange={(value) =>
                  filterDispatch({ type: "update:plan", value })
                }
              />
              <Separator />
              <RadioBox
                title={t("filter.sorts.sort")}
                icon={<ArrowDownNarrowWide />}
                prefix="sort"
                colLayout
                className={`mt-2`}
                items={[
                  // id-desc, id-asc
                  { id: "id-asc", value: t("filter.sorts.id-asc") },
                  { id: "id-desc", value: t("filter.sorts.id-desc") },

                  // quota-desc, quota-asc
                  { id: "quota-asc", value: t("filter.sorts.quota-asc") },
                  { id: "quota-desc", value: t("filter.sorts.quota-desc") },

                  // used-quota-desc, used-quota-asc
                  {
                    id: "used-quota-asc",
                    value: t("filter.sorts.used-quota-asc"),
                  },
                  {
                    id: "used-quota-desc",
                    value: t("filter.sorts.used-quota-desc"),
                  },

                  // plan-desc, plan-asc
                  { id: "plan-asc", value: t("filter.sorts.plan-asc") },
                  { id: "plan-desc", value: t("filter.sorts.plan-desc") },
                ]}
                value={filter.sort}
                onValueChange={(value) =>
                  filterDispatch({ type: "update:sort", value })
                }
              />
            </div>
            <DialogFooter>
              <DialogAction onClick={() => setFilterDialog(false)}>
                {t("confirm")}
              </DialogAction>
            </DialogFooter>
          </DialogContent>
        </Dialog>
        <Dialog
          open={createOpen}
          onOpenChange={(open) => {
            setCreateOpen(open);
            if (!open) resetCreateForm();
          }}
        >
          <DialogContent>
            <DialogTitle>{t("admin.add-user")}</DialogTitle>
            <DialogDescription>{t("admin.add-user-desc")}</DialogDescription>
            <div className="grid gap-3">
              <Input
                placeholder={t("admin.username")}
                value={createForm.username}
                onChange={(e) => updateCreateForm("username", e.target.value)}
              />
              <Input
                placeholder={t("admin.email")}
                value={createForm.email}
                onChange={(e) => updateCreateForm("email", e.target.value)}
              />
              <Input
                type="password"
                placeholder={t("auth.password")}
                value={createForm.password}
                onChange={(e) => updateCreateForm("password", e.target.value)}
                onKeyDown={async (e) => {
                  if (isEnter(e)) await createUser();
                }}
              />
            </div>
            <DialogFooter>
              <DialogCancel>{t("cancel")}</DialogCancel>
              <DialogAction onClick={createUser}>
                {t("admin.add-user")}
              </DialogAction>
            </DialogFooter>
          </DialogContent>
        </Dialog>
        <Input
          className={`search`}
          placeholder={t("admin.search-username")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onKeyDown={async (e) => {
            if (isEnter(e)) await update();
          }}
        />
        <Button
          size={`icon`}
          classNameWrapper={`flex-shrink-0 ml-2`}
          onClick={update}
        >
          <Search className={`h-4 w-4`} />
        </Button>
        <Button
          className={`whitespace-nowrap`}
          classNameWrapper={`flex-shrink-0 ml-2`}
          onClick={() => setCreateOpen(true)}
        >
          <PlusCircle className={`h-4 w-4 mr-2`} />
          {t("admin.add-user")}
        </Button>
      </div>
      {(data.data && data.data.length > 0) || page > 0 ? (
        <>
          <Table>
            <TableHeader>
              <TableRow className={`select-none whitespace-nowrap`}>
                <TableHead className="w-10">
                  <Checkbox checked={allChecked} onCheckedChange={toggleAll} />
                </TableHead>
                <TableHead>ID</TableHead>
                <TableHead>{t("admin.username")}</TableHead>
                <TableHead>{t("admin.email")}</TableHead>
                <TableHead>{t("admin.quota")}</TableHead>
                <TableHead>{t("admin.used-quota")}</TableHead>
                <TableHead>{t("admin.is-subscribed")}</TableHead>
                <TableHead>{t("admin.level")}</TableHead>
                <TableHead>{t("admin.total-month")}</TableHead>
                <TableHead>{t("admin.expired-at")}</TableHead>
                <TableHead>{t("admin.is-banned")}</TableHead>
                <TableHead>{t("admin.is-admin")}</TableHead>
                <TableHead>{t("admin.action")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(data.data || []).map((user, idx) => (
                <TableRow
                  key={idx}
                  className={selected.has(user.id) ? "bg-muted/50" : ""}
                >
                  <TableCell>
                    <Checkbox
                      checked={selected.has(user.id)}
                      onCheckedChange={() => toggleOne(user.id)}
                    />
                  </TableCell>
                  <TableCell>{user.id}</TableCell>
                  <TableCell className={`whitespace-nowrap`}>
                    {user.username}
                  </TableCell>
                  <TableCell className={`whitespace-nowrap`}>
                    {user.email || "-"}
                  </TableCell>
                  <TableCell>{user.quota}</TableCell>
                  <TableCell>{user.used_quota}</TableCell>
                  <TableCell>
                    {t(user.is_subscribed.toString())}
                    <Tips
                      className={`inline-block`}
                      content={t("admin.is-subscribed-tips")}
                    />
                  </TableCell>
                  <TableCell className={`whitespace-nowrap`}>
                    <Badge variant={`outline`}>
                      {t(`admin.identity.${userTypeArray[user.level]}`)}
                    </Badge>
                  </TableCell>
                  <TableCell>{user.total_month}</TableCell>
                  <TableCell className={`whitespace-nowrap`}>
                    {user.expired_at || "-"}
                  </TableCell>
                  <TableCell>{t(user.is_banned.toString())}</TableCell>
                  <TableCell>{t(user.is_admin.toString())}</TableCell>
                  <TableCell>
                    <OperationMenu user={user} onRefresh={update} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <PaginationAction
            current={page}
            total={data.total}
            onPageChange={setPage}
            offset
          />
        </>
      ) : loading ? (
        <div className={`flex flex-col mb-4 mt-12 items-center`}>
          <Loader2 className={`w-6 h-6 inline-block animate-spin`} />
        </div>
      ) : (
        <div className={`empty`}>
          <p>{t("admin.empty")}</p>
        </div>
      )}
      <div className={`user-action`}>
        <div className={`grow`} />
        <Button variant={`outline`} size={`icon`} onClick={update}>
          <RotateCw className={`h-4 w-4`} />
        </Button>
      </div>
      <div
        className={`fixed bottom-0 left-1/2 -translate-x-1/2 z-50 transition-all duration-300 ease-in-out ${
          selected.size > 0
            ? "translate-y-0 opacity-100"
            : "translate-y-full opacity-0 pointer-events-none"
        }`}
      >
        <div className="flex items-center gap-2 bg-background border border-b-0 shadow-lg rounded-t-2xl px-4 py-3">
          <span className="text-sm font-medium mr-1">
            {t("admin.selected", { count: selected.size })}
          </span>
          <Button
            size="sm"
            variant="destructive"
            className={batchActionClass}
            onClick={() => executeBatch("ban")}
          >
            <Ban className="h-3.5 w-3.5 mr-1" />
            {t("admin.ban")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            className={batchActionClass}
            onClick={() => executeBatch("unban")}
          >
            {t("admin.unban")}
          </Button>
          <div className="flex items-center gap-1">
            <Input
              className={batchQuotaInputClass}
              value={batchQuota}
              onChange={(e) =>
                setBatchQuota(e.target.value.replace(/[^\d.]/g, ""))
              }
            />
            <Button
              size="sm"
              variant="outline"
              className={batchActionClass}
              onClick={() => executeBatch("add_quota", Number(batchQuota))}
            >
              <PlusCircle className="h-3.5 w-3.5 mr-1" />
              {t("admin.quota")}
            </Button>
          </div>
          <Button
            size="sm"
            variant="ghost"
            className={batchActionClass}
            onClick={() => setSelected(new Set())}
          >
            {t("cancel")}
          </Button>
        </div>
      </div>
    </div>
  );
}

export default UserTable;
