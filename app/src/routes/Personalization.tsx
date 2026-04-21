import "@/assets/pages/personalization-page.less";
import { useTranslation } from "react-i18next";
import { useDispatch, useSelector } from "react-redux";
import * as settings from "@/store/settings.ts";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select.tsx";
import { Input } from "@/components/ui/input.tsx";
import { Textarea } from "@/components/ui/textarea.tsx";
import { ScrollArea } from "@/components/ui/scroll-area.tsx";

type SelectOption = {
  value: string;
  label: string;
};

type InlineSelectItemProps = {
  label: string;
  description?: string;
  value: string;
  options: SelectOption[];
  onChange: (value: string) => void;
};

function InlineSelectItem({
  label,
  description,
  value,
  options,
  onChange,
}: InlineSelectItemProps) {
  return (
    <div className="pz-item">
      <div className="pz-item-copy">
        <span className="pz-item-label">{label}</span>
        {description && (
          <span className="pz-item-description">{description}</span>
        )}
      </div>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger className="pz-select">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map((opt) => (
            <SelectItem key={opt.value} value={opt.value}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

function Personalization() {
  const { t } = useTranslation();
  const dispatch = useDispatch();

  const personaStyle = useSelector(settings.personaStyleSelector);
  const personaWarmth = useSelector(settings.personaWarmthSelector);
  const personaEnthusiasm = useSelector(settings.personaEnthusiasmSelector);
  const personaLists = useSelector(settings.personaListsSelector);
  const personaEmoji = useSelector(settings.personaEmojiSelector);
  const personaCustomInstruction = useSelector(
    settings.personaCustomInstructionSelector,
  );
  const personaNickname = useSelector(settings.personaNicknameSelector);
  const personaOccupation = useSelector(settings.personaOccupationSelector);
  const personaAboutUser = useSelector(settings.personaAboutUserSelector);

  const styleOptions: SelectOption[] = [
    {
      value: "friendly",
      label: t("settings.personalization.options.style.friendly"),
    },
    {
      value: "professional",
      label: t("settings.personalization.options.style.professional"),
    },
    {
      value: "concise",
      label: t("settings.personalization.options.style.concise"),
    },
    {
      value: "direct",
      label: t("settings.personalization.options.style.direct"),
    },
    {
      value: "playful",
      label: t("settings.personalization.options.style.playful"),
    },
  ];

  const levelOptions: SelectOption[] = [
    {
      value: "default",
      label: t("settings.personalization.options.level.default"),
    },
    { value: "low", label: t("settings.personalization.options.level.low") },
    {
      value: "medium",
      label: t("settings.personalization.options.level.medium"),
    },
    { value: "high", label: t("settings.personalization.options.level.high") },
  ];

  const listOptions: SelectOption[] = [
    {
      value: "default",
      label: t("settings.personalization.options.list.default"),
    },
    {
      value: "minimal",
      label: t("settings.personalization.options.list.minimal"),
    },
    {
      value: "balanced",
      label: t("settings.personalization.options.list.balanced"),
    },
    {
      value: "structured",
      label: t("settings.personalization.options.list.structured"),
    },
  ];

  const emojiOptions: SelectOption[] = [
    {
      value: "default",
      label: t("settings.personalization.options.emoji.default"),
    },
    {
      value: "none",
      label: t("settings.personalization.options.emoji.none"),
    },
    {
      value: "light",
      label: t("settings.personalization.options.emoji.light"),
    },
    {
      value: "expressive",
      label: t("settings.personalization.options.emoji.expressive"),
    },
  ];

  return (
    <ScrollArea className="pz-page">
      <div className="pz-content">
        {/* ── 基本风格和语调 ── */}
        <div className="pz-card">
          <InlineSelectItem
            label={t("settings.personalization.base-style")}
            description={t("settings.personalization.base-style-tip")}
            value={personaStyle}
            options={styleOptions}
            onChange={(v) => dispatch(settings.setPersonaStyle(v))}
          />

          {/* ── 特征 ── */}
          <div className="pz-divider" />
          <div className="pz-section-header">
            <span className="pz-section-title">
              {t("settings.personalization.traits")}
            </span>
            <span className="pz-section-note">
              {t("settings.personalization.traits-tip")}
            </span>
          </div>

          <InlineSelectItem
            label={t("settings.personalization.warmth")}
            value={personaWarmth}
            options={levelOptions}
            onChange={(v) => dispatch(settings.setPersonaWarmth(v))}
          />
          <InlineSelectItem
            label={t("settings.personalization.enthusiasm")}
            value={personaEnthusiasm}
            options={levelOptions}
            onChange={(v) => dispatch(settings.setPersonaEnthusiasm(v))}
          />
          <InlineSelectItem
            label={t("settings.personalization.headings-lists")}
            value={personaLists}
            options={listOptions}
            onChange={(v) => dispatch(settings.setPersonaLists(v))}
          />
          <InlineSelectItem
            label={t("settings.personalization.emoji")}
            value={personaEmoji}
            options={emojiOptions}
            onChange={(v) => dispatch(settings.setPersonaEmoji(v))}
          />

          {/* ── 自定义指令 ── */}
          <div className="pz-divider" />
          <div className="pz-field">
            <span className="pz-field-label">
              {t("settings.personalization.custom-instruction")}
            </span>
            <Textarea
              rows={3}
              value={personaCustomInstruction}
              placeholder={t(
                "settings.personalization.custom-instruction-placeholder",
              )}
              className="pz-textarea"
              onChange={(e) =>
                dispatch(settings.setPersonaCustomInstruction(e.target.value))
              }
            />
          </div>
        </div>

        {/* ── 关于你 ── */}
        <div className="pz-card">
          <div className="pz-section-header pz-section-header-top">
            <span className="pz-section-title">
              {t("settings.personalization.about-you")}
            </span>
          </div>

          <div className="pz-field">
            <span className="pz-field-label">
              {t("settings.personalization.nickname")}
            </span>
            <Input
              value={personaNickname}
              placeholder={t(
                "settings.personalization.nickname-placeholder",
              )}
              className="pz-input"
              onChange={(e) =>
                dispatch(settings.setPersonaNickname(e.target.value))
              }
            />
          </div>

          <div className="pz-field">
            <span className="pz-field-label">
              {t("settings.personalization.occupation")}
            </span>
            <Input
              value={personaOccupation}
              placeholder={t(
                "settings.personalization.occupation-placeholder",
              )}
              className="pz-input"
              onChange={(e) =>
                dispatch(settings.setPersonaOccupation(e.target.value))
              }
            />
          </div>

          <div className="pz-field">
            <div className="pz-field-copy">
              <span className="pz-field-label">
                {t("settings.personalization.about-user")}
              </span>
              <span className="pz-field-note">
                {t("settings.personalization.about-user-tip")}
              </span>
            </div>
            <Textarea
              rows={4}
              value={personaAboutUser}
              placeholder={t(
                "settings.personalization.about-user-placeholder",
              )}
              className="pz-textarea"
              onChange={(e) =>
                dispatch(settings.setPersonaAboutUser(e.target.value))
              }
            />
          </div>
        </div>
      </div>
    </ScrollArea>
  );
}

export default Personalization;
