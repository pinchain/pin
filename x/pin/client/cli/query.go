package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"

	"github.com/pinchain/pinchain/x/pin/types"
)

// GetQueryCmd returns the query commands of the x/pin module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the pin module",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdResolvePIN(),
		CmdPINsByOwner(),
		CmdListPINs(),
		CmdQueryParams(),
		CmdGeneratePIN(),
	)

	return cmd
}

// CmdResolvePIN resolves a PIN to its record.
func CmdResolvePIN() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resolve [pin]",
		Short:   "Resolve a PIN to its owner address, creation height and status",
		Example: "pinchaind query pin resolve A7K-92XM --output json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pin, err := types.NormalizeAndValidatePIN(args[0])
			if err != nil {
				return err
			}

			res, err := types.NewQueryClient(clientCtx).PIN(cmd.Context(), &types.QueryPINRequest{Pin: pin})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdPINsByOwner lists the PINs owned by an address.
func CmdPINsByOwner() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "by-owner [address]",
		Short: "List the PINs owned by an address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).PINsByOwner(
				cmd.Context(), &types.QueryPINsByOwnerRequest{Owner: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdListPINs lists all registered PINs.
func CmdListPINs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all registered PINs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).PINs(
				cmd.Context(), &types.QueryPINsRequest{Pagination: pageReq})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddPaginationFlagsToCmd(cmd, "pins")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryParams shows the module parameters.
func CmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Show the x/pin module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdGeneratePIN generates a candidate PIN locally (no network access).
func CmdGeneratePIN() *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate a random candidate PIN using cryptographically secure randomness",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pin, err := types.GeneratePIN()
			if err != nil {
				return err
			}
			cmd.Println(pin)
			return nil
		},
	}
}
