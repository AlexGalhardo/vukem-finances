import { cva, type VariantProps } from "class-variance-authority";
import { Tabs as TabsPrimitive } from "radix-ui";
import * as React from "react";
import { cn } from "../../lib/utils";

type TabsVariant = "line" | "segmented";

const TabsVariantContext = React.createContext<TabsVariant>("line");

function Tabs({ className, ...props }: React.ComponentProps<typeof TabsPrimitive.Root>) {
  return <TabsPrimitive.Root data-slot="tabs" className={cn("flex flex-col gap-3", className)} {...props} />;
}

const tabsListVariants = cva("inline-flex w-fit max-w-full items-center overflow-x-auto", {
  variants: {
    variant: {
      line: "gap-1 border-b border-border",
      segmented: "gap-1 rounded-md bg-muted p-1",
    },
  },
  defaultVariants: { variant: "line" },
});

function TabsList({
  className,
  variant = "line",
  ...props
}: React.ComponentProps<typeof TabsPrimitive.List> & VariantProps<typeof tabsListVariants>) {
  const resolved: TabsVariant = variant ?? "line";
  return (
    <TabsVariantContext.Provider value={resolved}>
      <TabsPrimitive.List
        data-slot="tabs-list"
        data-variant={resolved}
        className={cn(tabsListVariants({ variant: resolved }), className)}
        {...props}
      />
    </TabsVariantContext.Provider>
  );
}

function TabsTrigger({ className, ...props }: React.ComponentProps<typeof TabsPrimitive.Trigger>) {
  const variant = React.useContext(TabsVariantContext);
  return (
    <TabsPrimitive.Trigger
      data-slot="tabs-trigger"
      className={cn(
        "inline-flex h-9 shrink-0 items-center justify-center px-3 text-sm font-medium whitespace-nowrap text-muted-foreground transition-colors duration-(--vf-duration-fast) pointer-coarse:min-h-11 hover:text-foreground disabled:pointer-events-none disabled:text-disabled data-[state=active]:text-foreground",
        variant === "line" && "-mb-px border-b-2 border-transparent data-[state=active]:border-brand",
        variant === "segmented" &&
          "rounded-sm border border-transparent data-[state=active]:border-border data-[state=active]:bg-background",
        className,
      )}
      {...props}
    />
  );
}

function TabsContent({ className, ...props }: React.ComponentProps<typeof TabsPrimitive.Content>) {
  return <TabsPrimitive.Content data-slot="tabs-content" className={cn("outline-none", className)} {...props} />;
}

export { Tabs, TabsContent, TabsList, TabsTrigger };
