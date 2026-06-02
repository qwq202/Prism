import React from "react";

type IconElementProps = React.SVGProps<SVGSVGElement> & {
  className?: string;
  id?: string;
};

type IconProps = {
  icon?: React.ReactNode;
  className?: string;
  id?: string;
} & React.SVGProps<SVGSVGElement>;

function Icon({ icon, className, id, ...props }: IconProps) {
  if (!React.isValidElement<IconElementProps>(icon)) return null;

  return React.cloneElement(icon, {
    className: className,
    id: id,
    ...props,
  });
}

export default Icon;
