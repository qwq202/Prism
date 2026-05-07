export default function PrismLogo({
  className,
  onClick,
}: {
  className?: string;
  onClick?: () => void;
}) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 64 64"
      className={className}
      onClick={onClick}
      aria-label="Prism"
    >
      <text
        x="10"
        y="50"
        fontFamily="Georgia, 'Times New Roman', serif"
        fontSize="52"
        fontWeight="bold"
        letterSpacing="-2"
        className="fill-black dark:fill-white"
      >
        P
      </text>
      <circle cx="50" cy="48" r="7" fill="#3b82f6" />
    </svg>
  );
}
