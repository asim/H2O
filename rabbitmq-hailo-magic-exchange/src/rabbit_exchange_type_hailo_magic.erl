%% The contents of this file are subject to the Mozilla Public License
%% Version 1.1 (the "License"); you may not use this file except in
%% compliance with the License. You may obtain a copy of the License
%% at http://www.mozilla.org/MPL/
%%
%% Software distributed under the License is distributed on an "AS IS"
%% basis, WITHOUT WARRANTY OF ANY KIND, either express or implied. See
%% the License for the specific language governing rights and
%% limitations under the License.
%%
%% The Original Code is RabbitMQ.
%%
%% The Initial Developer of the Original Code is VMware, Inc.
%% Copyright (c) 2007-2011 VMware, Inc.  All rights reserved.
%%

-module(rabbit_exchange_type_hailo_magic).

-include_lib("rabbit_common/include/rabbit.hrl").
-include_lib("rabbit_common/include/rabbit_framing.hrl").
-include_lib("stdlib/include/qlc.hrl").

-behaviour(rabbit_exchange_type).

-export([description/0, serialise_events/0, route/2]).
-export([validate/1, validate_binding/2,
         create/2, delete/3, policy_changed/2, add_binding/3,
         remove_bindings/3, assert_args_equivalence/2]).

-rabbit_boot_step({?MODULE,
                   [{description, "exchange type headers"},
                    {mfa,         {rabbit_registry, register,
                                   [exchange, <<"x-hailo2">>, ?MODULE]}},
                    {requires,    rabbit_registry},
                    {enables,     kernel_ready}]}).

-ifdef(use_specs).
-spec(headers_match/2 :: (rabbit_framing:amqp_table(),
                          rabbit_framing:amqp_table()) -> boolean()).
-endif.

description() ->
    [{name, <<"x-hailo2">>},
     {description, <<"Magic hailo glue">>}].

serialise_events() -> false.

route(#exchange{name = Name},
      #delivery{message = #basic_message{content = Content}}) ->
    %rabbit_log:info("Headers ~p",[(Content#content.properties)#'P_basic'.headers]),

    Headers = case (Content#content.properties)#'P_basic'.headers of
                  undefined -> [];
                  H         -> rabbit_misc:sort_field_table(H)
              end,
    {_,Service} = table_lookup(Headers, <<"service">>),
    RecFrom = table_lookup(Headers, <<"x-received-from">>),
    Label = case table_lookup(Headers, <<"x-label">>) of
              {_, Value} -> Value;
              undefined -> <<"default">>
            end,
    Matches = [ {M, Args} ||
                    {M, Args} <- find_bindings(Name, Service),
                    weight(Args) > 0,
                    check_federation(RecFrom, Args)
              ],
    case filter_by_label(Label, Matches) of
      [] -> random_weighted_pick(filter_by_label(<<"default">>, Matches));
      LabelMatches -> random_weighted_pick(LabelMatches)
    end.

filter_by_label(Label, Matches) ->
  lists:filter(fun ({M, Args}) ->
    case table_lookup(Args, <<"x-label">>) of
      {_, Value} -> Label == Value;
      undefined  -> Label == <<"default">>
    end
  end, Matches).

check_federation(undefined, _) -> true;
check_federation(_, Args) ->
  parse_x_nofed(table_lookup(Args, <<"x-nofed">>)).

sum_weights(Matches) ->
  lists:foldl(fun({_, Args}, Sum) -> Sum+weight(Args) end, 0, Matches).

random_weighted_pick(Matches) ->
  case sum_weights(Matches) of
    0     -> [];
    Total -> random_weighted_pick(Matches, random:uniform(Total))
  end.

random_weighted_pick([], _) -> [];

random_weighted_pick([{M, Args}|T], Total) ->
  W = weight(Args),
  NewTotal = Total - W,
  if
    NewTotal > 0 -> random_weighted_pick(T, NewTotal);
    true         -> [M]
  end.

weight(Args) -> parse_x_weight(table_lookup(Args, <<"x-weight">>)).

validate_binding(_X, #binding{args = Args}) ->
    case table_lookup(Args, <<"x-weight">>) of
        {long, _}       -> ok;
        {WType, WOther}     -> {error,
                                 {binding_invalid,
                                  "Invalid x-weight field type ~p (value ~p); "
                                  "expected long", [WType, WOther]}};
        undefined         -> ok %% [0]
    end.


parse_x_weight({long, Weight}) -> Weight;
parse_x_weight(_) -> 1.

parse_x_nofed({longstr, <<"yes">>}) -> true;
parse_x_nofed(_) -> false.

validate(_X) -> ok.
create(_Tx, _X) -> ok.
delete(_Tx, _X, _Bs) -> ok.
policy_changed(_X1, _X2) -> ok.
add_binding(_Tx, _X, _B) -> ok.
remove_bindings(_Tx, _X, _Bs) -> ok.
assert_args_equivalence(X, Args) ->
    rabbit_exchange:assert_args_equivalence(X, Args).
    

table_lookup(Table, Key) ->
    case lists:keyfind(Key, 1, Table) of
        {_, TypeBin, ValueBin} -> {TypeBin, ValueBin};
        false                  -> undefined
    end.

find_bindings(SrcName, RoutingKey) ->
    Routes = find_routes(#route{binding = #binding{source = SrcName,
                                          destination = '_',
                                          key = RoutingKey,
                                          _ = '_'}},
                []),
    [{DestinationName, Args} || #route{binding = Binding = #binding{_ = _,
                                          destination = DestinationName,
                                          _ = _,
                                          args = Args}} <- Routes].

%%--------------------------------------------------------------------

%% Normally we'd call mnesia:dirty_select/2 here, but that is quite
%% expensive for the same reasons as above, and, additionally, due to
%% mnesia 'fixing' the table with ets:safe_fixtable/2, which is wholly
%% unnecessary. According to the ets docs (and the code in erl_db.c),
%% 'select' is safe anyway ("Functions that internally traverse over a
%% table, like select and match, will give the same guarantee as
%% safe_fixtable.") and, furthermore, even the lower level iterators
%% ('first' and 'next') are safe on ordered_set tables ("Note that for
%% tables of the ordered_set type, safe_fixtable/2 is not necessary as
%% calls to first/1 and next/2 will always succeed."), which
%% rabbit_route is.
find_routes(MatchHead, Conditions) ->
    ets:select(rabbit_route, [{MatchHead, Conditions, ['$_']}]).
